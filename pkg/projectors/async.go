/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/untillpro/goutils/iterate"
	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	imetrics "github.com/voedger/voedger/pkg/metrics"
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

type (
	plogEvent struct {
		istructs.Offset
		istructs.IPLogEvent
	}
	plogBatch []plogEvent

	// Should return:
	//	- events in batch and no error, if events succussfully read
	//	- no events (empty batch) and no error, if all events are read
	//	- no events (empty  batch) and error, if error occurs while reading PLog.
	// Should not return simultaneously any events (non empty batch) and error
	readPLogBatch func(*plogBatch) error
)

// implements ServiceOperator
type asyncActualizer struct {
	conf         AsyncActualizerConf
	factory      istructs.ProjectorFactory
	pipeline     pipeline.IAsyncPipeline
	structs      istructs.IAppStructs
	offset       istructs.Offset
	name         string
	readCtx      *asyncActualizerContextState
	projErrState int32 // 0 - no error, 1 - error
	plogBatch          // [50]plogEvent
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
	if a.conf.FlushPositionInterval == 0 {
		a.conf.FlushPositionInterval = defaultFlushPositionInterval
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
	if err = a.waitForAppDeploy(ctx); err != nil {
		panic(err)
	}
	for ctx.Err() == nil {
		if err = a.init(ctx); err == nil {
			logger.Trace(a.name, "started")
			if err = a.keepReading(); err != nil {
				a.conf.LogError(a.name, err)
			}
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

func (a *asyncActualizer) waitForAppDeploy(ctx context.Context) error {
	for ctx.Err() == nil {
		ap, err := a.conf.AppPartitions.Borrow(a.conf.AppQName, a.conf.Partition, cluster.ProcessorKind_Actualizer)
		if err == nil || errors.Is(err, appparts.ErrNotAvailableEngines) {
			if ap != nil {
				ap.Release()
			}
			return nil
		}
		if !errors.Is(err, appparts.ErrNotFound) {
			return err
		}
		time.Sleep(borrowRetryDelay)
	}
	return nil // consider "context canceled" as expected error
}

func (a *asyncActualizer) init(ctx context.Context) (err error) {
	a.plogBatch = make(plogBatch, 0, plogReadBatchSize)

	a.structs = a.conf.AppStructs() // TODO: must be borrowed and finally released
	a.readCtx = &asyncActualizerContextState{}

	a.readCtx.ctx, a.readCtx.cancel = context.WithCancel(ctx)

	projector := a.factory(a.conf.Partition)
	iProjector := a.structs.AppDef().Projector(projector.Name)
	if iProjector == nil {
		return fmt.Errorf("async projector %s is not defined in AppDef", projector.Name)
	}

	// https://github.com/voedger/voedger/issues/1048
	hasIntentsExceptViewAndRecord, _ := iterate.FindFirst(iProjector.Intents().Enum, func(storage appdef.IStorage) bool {
		n := storage.Name()
		return n != state.View && n != state.Record
	})
	// https://github.com/voedger/voedger/issues/1092
	hasStatesExceptViewAndRecord, _ := iterate.FindFirst(iProjector.States().Enum, func(storage appdef.IStorage) bool {
		n := storage.Name()
		return n != state.View && n != state.Record
	})
	nonBuffered := hasIntentsExceptViewAndRecord || hasStatesExceptViewAndRecord
	p := &asyncProjector{
		partition:             a.conf.Partition,
		aametrics:             a.conf.AAMetrics,
		flushPositionInterval: a.conf.FlushPositionInterval,
		lastSave:              time.Now(),
		projErrState:          &a.projErrState,
		metrics:               a.conf.Metrics,
		vvmName:               a.conf.VvmName,
		appQName:              a.conf.AppQName,
		projector:             projector,
		iProjector:            iProjector,
		nonBuffered:           nonBuffered,
	}

	if p.metrics != nil {
		p.projInErrAddr = p.metrics.AppMetricAddr(ProjectorsInError, a.conf.VvmName, a.conf.AppQName)
	}

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
		p.EventProvider,
		a.conf.IntentsLimit,
		a.conf.BundlesLimit,
		a.conf.Opts...)

	a.name = fmt.Sprintf("%s [%d]", p.projector.Name, a.conf.Partition)

	projectorOp := pipeline.WireAsyncOperator("Projector", p, a.conf.FlushInterval)

	errHandler := &asyncErrorHandler{
		readCtx:      a.readCtx,
		projErrState: &a.projErrState,
		metrics:      a.conf.Metrics,
		vvmName:      a.conf.VvmName,
		appQName:     a.conf.AppQName,
	}
	errorHandlerOp := pipeline.WireAsyncOperator("ErrorHandler", errHandler)

	a.pipeline = pipeline.NewAsyncPipeline(ctx, a.name, projectorOp, errorHandlerOp)

	if a.conf.channel, err = a.conf.Broker.NewChannel(istructs.SubjectLogin(a.name), n10nChannelDuration); err != nil {
		return err
	}
	return a.conf.Broker.Subscribe(a.conf.channel, in10n.ProjectionKey{
		App:        a.conf.AppQName,
		Projection: PLogUpdatesQName,
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
	err = a.readPlogToTheEnd(a.readCtx.ctx)
	if err != nil {
		a.cancelChannel(err)
		return
	}
	a.conf.Broker.WatchChannel(a.readCtx.ctx, a.conf.channel, func(projection in10n.ProjectionKey, offset istructs.Offset) {
		if logger.IsTrace() {
			logger.Trace(fmt.Sprintf("%s received n10n: offset %d, last handled: %d", a.name, offset, a.offset))
		}
		if a.offset < offset {
			err = a.readPlogToOffset(a.readCtx.ctx, offset)
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

func (a *asyncActualizer) readPlogByBatches(readBatch readPLogBatch) error {
	for a.readCtx.ctx.Err() == nil {
		if err := readBatch(&a.plogBatch); err != nil {
			return err
		}
		if len(a.plogBatch) == 0 {
			break
		}
		for _, e := range a.plogBatch {
			if err := a.handleEvent(e.Offset, e.IPLogEvent); err != nil {
				return err
			}
			if a.readCtx.ctx.Err() != nil {
				return nil // canceled
			}
		}
	}
	return nil
}

func (a *asyncActualizer) borrowAppPart(ctx context.Context) (ap appparts.IAppPartition, err error) {
	for ctx.Err() == nil {
		if ap, err = a.conf.AppPartitions.Borrow(a.conf.AppQName, a.conf.Partition, cluster.ProcessorKind_Actualizer); err == nil {
			return ap, nil
		}
		if errors.Is(err, appparts.ErrNotAvailableEngines) {
			time.Sleep(borrowRetryDelay)
			continue
		}
		return nil, err
	}
	return nil, ctx.Err()
}

func (a *asyncActualizer) readPlogToTheEnd(ctx context.Context) error {
	return a.readPlogByBatches(func(batch *plogBatch) (err error) {
		*batch = (*batch)[:0]

		ap, err := a.borrowAppPart(ctx)
		if err != nil {
			return err
		}

		defer ap.Release()

		err = ap.AppStructs().Events().ReadPLog(a.readCtx.ctx, a.conf.Partition, a.offset+1, istructs.ReadToTheEnd,
			func(ofs istructs.Offset, event istructs.IPLogEvent) error {
				if *batch = append(*batch, plogEvent{ofs, event}); len(*batch) == cap(*batch) {
					return errBatchFull
				}
				return nil
			})
		if len(*batch) > 0 {
			//nolint: suppress error if at least one event was read
			return nil
		}
		return err
	})
}

func (a *asyncActualizer) readPlogToOffset(ctx context.Context, tillOffset istructs.Offset) error {
	return a.readPlogByBatches(func(batch *plogBatch) (err error) {
		*batch = (*batch)[:0]

		ap, err := a.borrowAppPart(ctx)
		if err != nil {
			return err
		}

		defer ap.Release()

		plog := ap.AppStructs().Events()
		for readOffset := a.offset + 1; readOffset <= tillOffset; readOffset++ {
			if err = plog.ReadPLog(a.readCtx.ctx, a.conf.Partition, readOffset, 1,
				func(ofs istructs.Offset, event istructs.IPLogEvent) error {
					if *batch = append(*batch, plogEvent{ofs, event}); len(*batch) == cap(*batch) {
						return errBatchFull
					}
					return nil
				}); err != nil {
				break
			}
		}
		if len(*batch) > 0 {
			//nolint: suppress error if at least one event was read
			return nil
		}
		return err
	})
}

func (a *asyncActualizer) readOffset(projectorName appdef.QName) (err error) {
	a.offset, err = ActualizerOffset(a.structs, a.conf.Partition, projectorName)
	return
}

type asyncProjector struct {
	pipeline.AsyncNOOP
	state                 state.IBundledHostState
	partition             istructs.PartitionID
	event                 istructs.IPLogEvent
	projector             istructs.Projector
	pLogOffset            istructs.Offset
	aametrics             AsyncActualizerMetrics
	metrics               imetrics.IMetrics
	projInErrAddr         *imetrics.MetricValue
	flushPositionInterval time.Duration
	acceptedSinceSave     bool
	lastSave              time.Time
	projErrState          *int32
	vvmName               string
	appQName              istructs.AppQName
	iProjector            appdef.IProjector
	nonBuffered           bool
}

func (p *asyncProjector) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	defer work.Release()
	w := work.(*workpiece)

	p.event = w.event
	p.pLogOffset = w.pLogOffset
	if p.aametrics != nil {
		p.aametrics.Set(aaCurrentOffset, p.partition, p.projector.Name, float64(w.pLogOffset))
	}

	triggeringQNames := p.iProjector.EventsMap()
	if isAcceptable(w.event, p.iProjector.WantErrors(), triggeringQNames, p.iProjector.App()) {
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

		if readyToFlushBundle || p.nonBuffered {
			return nil, p.flush()
		}
	}

	return nil, err
}
func (p *asyncProjector) Flush(_ pipeline.OpFuncFlush) (err error) { return p.flush() }
func (p *asyncProjector) WSIDProvider() istructs.WSID              { return p.event.Workspace() }
func (p *asyncProjector) EventProvider() istructs.IPLogEvent       { return p.event }
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
	if p.aametrics != nil {
		p.aametrics.Increase(aaFlushesTotal, p.partition, p.projector.Name, 1)
		p.aametrics.Set(aaStoredOffset, p.partition, p.projector.Name, float64(p.pLogOffset))
	}
	err = p.state.FlushBundles()
	if err == nil && p.projInErrAddr != nil {
		if atomic.CompareAndSwapInt32(p.projErrState, 1, 0) {
			p.projInErrAddr.Increase(-1)
		} else { // state not changed
			p.projInErrAddr.Increase(0)
		}
	}
	return err
}

type asyncErrorHandler struct {
	pipeline.AsyncNOOP
	readCtx      *asyncActualizerContextState
	metrics      imetrics.IMetrics
	vvmName      string
	appQName     istructs.AppQName
	projErrState *int32
}

func (h *asyncErrorHandler) OnError(_ context.Context, err error) {
	if atomic.CompareAndSwapInt32(h.projErrState, 0, 1) {
		if h.metrics != nil {
			h.metrics.IncreaseApp(ProjectorsInError, h.vvmName, h.appQName, 1)
		}
	}
	h.readCtx.cancelWithError(err)
}

func ActualizerOffset(appStructs istructs.IAppStructs, partition istructs.PartitionID, projectorName appdef.QName) (offset istructs.Offset, err error) {
	key := appStructs.ViewRecords().KeyBuilder(qnameProjectionOffsets)
	key.PutInt32(partitionFld, int32(partition))
	key.PutQName(projectorNameFld, projectorName)
	value, err := appStructs.ViewRecords().Get(istructs.NullWSID, key)
	if errors.Is(err, istructsmem.ErrRecordNotFound) {
		return istructs.NullOffset, nil
	}
	if err != nil {
		return istructs.NullOffset, err
	}
	return istructs.Offset(value.AsInt64(offsetFld)), err
}
