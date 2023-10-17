/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"fmt"
	"time"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
)

type workpiece struct {
	event      istructs.IPLogEvent
	pLogOffset istructs.Offset
}

func (w *workpiece) Release() {
	w.event.Release()
}

// implements ServiceOperator
type asyncActualizer struct {
	conf     AsyncActualizerConf
	factory  istructs.ProjectorFactory
	pipeline pipeline.IAsyncPipeline
	structs  istructs.IAppStructs
	offset   istructs.Offset
	name     string
	readCtx  *asyncActualizerContextState
}

func (a *asyncActualizer) Prepare(interface{}) error {
	if a.conf.IntentsLimit == 0 {
		a.conf.IntentsLimit = defaultIntentsLimit
	}

	if a.conf.BundlesLimit == 0 {
		a.conf.BundlesLimit = defaultBundlesLimit
	}

	if a.conf.FlushInterval == 0 {
		a.conf.FlushInterval = defaultFlushInterval
	}
	if a.conf.FlushPositionInverval == 0 {
		a.conf.FlushPositionInverval = defaultFlushPositionInterval
	}
	if a.conf.AfterError == nil {
		a.conf.AfterError = time.After
	}

	if a.conf.LogError == nil {
		a.conf.LogError = logger.Error
	}
	return nil
}
func (a *asyncActualizer) Run(ctx context.Context) {
	var err error
	for ctx.Err() == nil {
		if err = a.init(ctx); err == nil {
			logger.Trace(a.name, "started")
			err = a.keepReading()
			a.conf.LogError(a.name, err)
		}
		a.finit() // even execute if a.init has failed
		if ctx.Err() == nil && err != nil {
			a.conf.LogError(a.name, err)
			select {
			case <-ctx.Done():
			case <-a.conf.AfterError(actualizerErrorDelay):
			}
		}
	}
}
func (a *asyncActualizer) Stop() {}
func (a *asyncActualizer) cancelChannel(e error) {
	a.readCtx.cancelWithError(e)
	a.conf.Broker.WatchChannel(a.readCtx.ctx, a.conf.channel, func(projection in10n.ProjectionKey, offset istructs.Offset) {})
}

func (a *asyncActualizer) init(ctx context.Context) (err error) {
	a.structs = a.conf.AppStructs()
	a.readCtx = &asyncActualizerContextState{}

	a.readCtx.ctx, a.readCtx.cancel = context.WithCancel(ctx)

	p := &asyncProjector{partition: a.conf.Partition, metrics: a.conf.Metrics, flushPositionInterval: a.conf.FlushPositionInverval, lastSave: time.Now()}
	p.projector = a.factory(a.conf.Partition)

	err = a.readOffset(p.projector.Name)
	if err != nil {
		a.conf.LogError(a.name, err)
		return err
	}

	p.state = state.ProvideAsyncActualizerStateFactory()(
		ctx,
		a.structs,
		state.SimplePartitionIDFunc(a.conf.Partition),
		p.WSIDProvider,
		func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {
			a.conf.Broker.Update(in10n.ProjectionKey{
				App:        a.conf.AppQName,
				Projection: view,
				WS:         wsid,
			}, offset)
		},
		a.conf.SecretReader,
		a.conf.IntentsLimit,
		a.conf.BundlesLimit,
		a.conf.Opts...)

	a.name = fmt.Sprintf("%s [%d]", p.projector.Name, a.conf.Partition)

	projectorOp := pipeline.WireAsyncOperator("Projector", p, a.conf.FlushInterval)

	errorHandlerOp := pipeline.WireAsyncOperator("ErrorHandler", &asyncErrorHandler{readCtx: a.readCtx})

	a.pipeline = pipeline.NewAsyncPipeline(ctx, a.name, projectorOp, errorHandlerOp)

	if a.conf.channel, err = a.conf.Broker.NewChannel(istructs.SubjectLogin(a.name), n10nChannelDuration); err != nil {
		return err
	}
	return a.conf.Broker.Subscribe(a.conf.channel, in10n.ProjectionKey{
		App:        a.conf.AppQName,
		Projection: PlogQName,
		WS:         istructs.WSID(a.conf.Partition),
	})
}

func (a *asyncActualizer) finit() {
	if a.pipeline != nil {
		a.pipeline.Close()
	}
	if logger.IsTrace() {
		logger.Trace(fmt.Sprintf("%s finalized", a.name))
	}
}

func (a *asyncActualizer) keepReading() (err error) {
	err = a.readPlogToTheEnd()
	if err != nil {
		a.cancelChannel(err)
		return
	}
	a.conf.Broker.WatchChannel(a.readCtx.ctx, a.conf.channel, func(projection in10n.ProjectionKey, offset istructs.Offset) {
		if logger.IsTrace() {
			logger.Trace(fmt.Sprintf("%s received n10n: offset %d, last handled: %d", a.name, offset, a.offset))
		}
		if a.offset < offset {
			err = a.readPlogToTheEnd2(offset)
			if err != nil {
				a.conf.LogError(a.name, err)
				a.readCtx.cancelWithError(err)
			}
		}
	})
	return a.readCtx.error()
}

func (a *asyncActualizer) handleEvent(pLogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
	work := &workpiece{
		event:      event,
		pLogOffset: pLogOffset,
	}

	err = a.pipeline.SendAsync(work)
	if err != nil {
		a.conf.LogError(a.name, err)
		return
	}

	a.offset = pLogOffset

	if logger.IsTrace() {
		logger.Trace(fmt.Sprintf("offset %d for %s", a.offset, a.name))
	}

	return
}

func (a *asyncActualizer) readPlogToTheEnd() (err error) {
	return a.conf.AppStructs().Events().ReadPLog(a.readCtx.ctx, a.conf.Partition, a.offset+1, istructs.ReadToTheEnd, a.handleEvent)
}

func (a *asyncActualizer) readPlogToTheEnd2(tillOffset istructs.Offset) (err error) {
	for readOffset := a.offset + 1; readOffset <= tillOffset; readOffset++ {
		if err = a.conf.AppStructs().Events().ReadPLog(a.readCtx.ctx, a.conf.Partition, readOffset, 1, a.handleEvent); err != nil {
			return
		}
	}
	return nil
}

func (a *asyncActualizer) readOffset(projectorName appdef.QName) (err error) {
	a.offset, err = ActualizerOffset(a.structs, a.conf.Partition, projectorName)
	return
}

type asyncProjector struct {
	pipeline.AsyncNOOP
	state                 state.IBundledHostState
	partition             istructs.PartitionID
	wsid                  istructs.WSID
	projector             istructs.Projector
	pLogOffset            istructs.Offset
	metrics               AsyncActualizerMetrics
	flushPositionInterval time.Duration
	acceptedSinceSave     bool
	lastSave              time.Time
}

func (p *asyncProjector) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	defer work.Release()
	w := work.(*workpiece)

	p.wsid = w.event.Workspace()
	p.pLogOffset = w.pLogOffset
	if p.metrics != nil {
		p.metrics.Set(aaCurrentOffset, p.partition, p.projector.Name, float64(w.pLogOffset))
	}

	if isAcceptable(p.projector, w.event) {
		err = p.projector.Func(w.event, p.state, p.state)
		if err != nil {
			return nil, fmt.Errorf("wsid[%d] offset[%d]: %w", w.event.Workspace(), w.event.WLogOffset(), err)
		}
		if logger.IsVerbose() {
			logger.Verbose(fmt.Sprintf("%s: handled %d", p.projector.Name, p.pLogOffset))
		}

		p.acceptedSinceSave = true

		readyToFlushBundle, err := p.state.ApplyIntents()
		if err != nil {
			return nil, err
		}

		if readyToFlushBundle || p.projector.NonBuffered {
			return nil, p.flush()
		}
	}

	return nil, err
}
func (p *asyncProjector) Flush(_ pipeline.OpFuncFlush) (err error) { return p.flush() }
func (p *asyncProjector) WSIDProvider() istructs.WSID              { return p.wsid }
func (p *asyncProjector) savePosition() error {
	defer func() {
		p.acceptedSinceSave = false
		p.lastSave = time.Now()
	}()
	key, e := p.state.KeyBuilder(state.View, qnameProjectionOffsets)
	if e != nil {
		return e
	}
	key.PutInt64(state.Field_WSID, int64(istructs.NullWSID))
	key.PutInt32(partitionFld, int32(p.partition))
	key.PutQName(projectorNameFld, p.projector.Name)
	value, e := p.state.NewValue(key)
	if e != nil {
		return e
	}
	value.PutInt64(offsetFld, int64(p.pLogOffset))
	return nil
}
func (p *asyncProjector) flush() (err error) {
	if p.pLogOffset == istructs.NullOffset {
		return
	}
	defer func() {
		p.pLogOffset = istructs.NullOffset
	}()

	timeToSavePosition := time.Since(p.lastSave) >= p.flushPositionInterval
	if p.acceptedSinceSave || timeToSavePosition {
		if err = p.savePosition(); err != nil {
			return
		}
	}
	_, err = p.state.ApplyIntents()
	if err != nil {
		return err
	}
	if p.metrics != nil {
		p.metrics.Increase(aaFlushesTotal, p.partition, p.projector.Name, 1)
		p.metrics.Set(aaStoredOffset, p.partition, p.projector.Name, float64(p.pLogOffset))
	}
	return p.state.FlushBundles()
}

type asyncErrorHandler struct {
	pipeline.AsyncNOOP
	readCtx *asyncActualizerContextState
}

func (h *asyncErrorHandler) OnError(_ context.Context, err error) { h.readCtx.cancelWithError(err) }

func ActualizerOffset(appStructs istructs.IAppStructs, partition istructs.PartitionID, projectorName appdef.QName) (offset istructs.Offset, err error) {
	key := appStructs.ViewRecords().KeyBuilder(qnameProjectionOffsets)
	key.PutInt32(partitionFld, int32(partition))
	key.PutQName(projectorNameFld, projectorName)
	value, err := appStructs.ViewRecords().Get(istructs.NullWSID, key)
	if err == istructsmem.ErrRecordNotFound {
		return istructs.NullOffset, nil
	}
	if err != nil {
		return istructs.NullOffset, err
	}
	return istructs.Offset(value.AsInt64(offsetFld)), err
}
