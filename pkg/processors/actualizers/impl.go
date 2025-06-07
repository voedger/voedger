/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/stateprovide"
)

func syncActualizerFactory(conf SyncActualizerConf, projectors istructs.Projectors) pipeline.ISyncOperator {
	if conf.IntentsLimit == 0 {
		conf.IntentsLimit = defaultIntentsLimit
	}
	service := &eventService{}
	ss := make([]state.IHostState, 0, len(projectors))
	bb := make([]pipeline.ForkOperatorOptionFunc, 0, len(projectors))
	for _, p := range projectors {
		b, s := newSyncBranch(conf, p, service)
		ss = append(ss, s)
		bb = append(bb, b)
	}
	h := &syncErrorHandler{ss: ss}
	return pipeline.NewSyncPipeline(conf.Ctx, "PartitionSyncActualizer",
		pipeline.WireFunc("Update event", func(_ context.Context, work pipeline.IWorkpiece) (err error) {
			service.event = work.(interface{ Event() istructs.IPLogEvent }).Event()
			return nil
		}),
		pipeline.WireFunc("Update IAppStructs", func(_ context.Context, work pipeline.IWorkpiece) (err error) {
			service.appStructs = work.(interface{ AppPartition() appparts.IAppPartition }).AppPartition().AppStructs()
			return nil
		}),
		pipeline.WireSyncOperator("SyncActualizer", pipeline.ForkOperator(pipeline.ForkSame, bb[0], bb[1:]...)),
		pipeline.WireFunc("IntentsApplier", func(_ context.Context, _ pipeline.IWorkpiece) (err error) {
			for _, st := range ss {
				err = st.ApplyIntents()
				if err != nil {
					return
				}
			}
			return
		}),
		pipeline.WireSyncOperator("ErrorHandler", h))
}

func newSyncBranch(conf SyncActualizerConf, projector istructs.Projector, service *eventService) (fn pipeline.ForkOperatorOptionFunc, s state.IHostState) {
	pipelineName := fmt.Sprintf("[%d] %s", conf.Partition, projector.Name)
	s = stateprovide.ProvideSyncActualizerStateFactory()(
		conf.Ctx,
		service.getIAppStructs,
		state.SimplePartitionIDFunc(conf.Partition),
		service.getWSID,
		conf.N10nFunc,
		conf.SecretReader,
		service.getEvent,
		conf.IntentsLimit)
	fn = pipeline.ForkBranch(pipeline.NewSyncPipeline(conf.Ctx, pipelineName,
		pipeline.WireFunc("Projector",
			func(ctx context.Context, work pipeline.IWorkpiece) error {
				appPart := work.(interface{ AppPartition() appparts.IAppPartition }).AppPartition()
				appDef := appPart.AppStructs().AppDef()
				prj := appdef.Projector(appDef.Type, projector.Name)
				event := s.PLogEvent()
				if !ProjectorEvent(prj, event) {
					return nil
				}
				return appPart.Invoke(ctx, projector.Name, s, s)
			}),
		pipeline.WireFunc("IntentsValidator", func(_ context.Context, _ pipeline.IWorkpiece) (err error) {
			return s.ValidateIntents()
		})))
	return
}

type syncErrorHandler struct {
	pipeline.NOOP
	ss  []state.IHostState
	err error
}

func (h *syncErrorHandler) DoSync(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	if h.err != nil {
		for _, s := range h.ss {
			s.ClearIntents()
		}
		err = h.err
		h.err = nil
	}
	return
}

func (h *syncErrorHandler) OnErr(err error, _ interface{}, _ pipeline.IWorkpieceContext) error {
	h.err = err
	return nil
}

type eventService struct {
	event      istructs.IPLogEvent
	appStructs istructs.IAppStructs
}

func (s *eventService) getWSID() istructs.WSID { return s.event.Workspace() }

func (s *eventService) getEvent() istructs.IPLogEvent { return s.event }

func (s *eventService) getIAppStructs() istructs.IAppStructs { return s.appStructs }

func provideViewDefImpl(wsb appdef.IWorkspaceBuilder, qname appdef.QName, buildFunc ViewTypeBuilder) {
	builder := wsb.AddView(qname)
	if buildFunc != nil {
		buildFunc(builder)
	}
}
