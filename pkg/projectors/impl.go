/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	istructs "github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
)

func asyncActualizerFactory(conf AsyncActualizerConf, factory istructs.ProjectorFactory) (pipeline.ISyncOperator, error) {
	return pipeline.ServiceOperator(&asyncActualizer{
		factory: factory,
		conf:    conf,
	}), nil
}

func syncActualizerFactory(conf SyncActualizerConf, projection istructs.ProjectorFactory, otherProjections ...istructs.ProjectorFactory) pipeline.ISyncOperator {
	if conf.IntentsLimit == 0 {
		conf.IntentsLimit = defaultIntentsLimit
	}
	if conf.WorkToEvent == nil {
		conf.WorkToEvent = func(work interface{}) istructs.IPLogEvent { return work.(istructs.IPLogEvent) }
	}
	service := &eventService{}
	ss := make([]state.IHostState, 0, len(otherProjections)+1)
	bb := make([]pipeline.ForkOperatorOptionFunc, 0, len(otherProjections))
	b, s := newSyncBranch(conf, projection, service)
	bb = append(bb, b)
	ss = append(ss, s)
	for _, otherProjection := range otherProjections {
		b, s = newSyncBranch(conf, otherProjection, service)
		ss = append(ss, s)
		bb = append(bb, b)
	}
	h := &syncErrorHandler{ss: ss}
	return pipeline.NewSyncPipeline(conf.Ctx, "PartitionSyncActualizer",
		pipeline.WireFunc("Update event", func(_ context.Context, work interface{}) (err error) {
			service.event = conf.WorkToEvent(work)
			return err
		}),
		pipeline.WireSyncOperator("SyncActualizer", pipeline.ForkOperator(pipeline.ForkSame, bb[0], bb[1:]...)),
		pipeline.WireFunc("IntentsApplier", func(_ context.Context, _ interface{}) (err error) {
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

func newSyncBranch(conf SyncActualizerConf, projectorFactoy istructs.ProjectorFactory, service *eventService) (fn pipeline.ForkOperatorOptionFunc, s state.IHostState) {
	projector := projectorFactoy(conf.Partition)
	pipelineName := fmt.Sprintf("[%d] %s", conf.Partition, projector.Name)
	s = state.ProvideSyncActualizerStateFactory()(
		conf.Ctx,
		conf.AppStructs(),
		state.SimplePartitionIDFunc(conf.Partition),
		service.getWSID,
		conf.N10nFunc,
		conf.SecretReader,
		conf.IntentsLimit)
	fn = pipeline.ForkBranch(pipeline.NewSyncPipeline(conf.Ctx, pipelineName,
		pipeline.WireFunc("Projector", func(_ context.Context, _ interface{}) (err error) {
			if !isAcceptable(projector, service.event) {
				return err
			}
			return projector.Func(service.event, s, s)
		}),
		pipeline.WireFunc("IntentsValidator", func(_ context.Context, _ interface{}) (err error) {
			return s.ValidateIntents()
		})))
	return
}

type syncErrorHandler struct {
	pipeline.NOOP
	ss  []state.IHostState
	err error
}

func (h *syncErrorHandler) DoSync(_ context.Context, _ interface{}) (err error) {
	if h.err != nil {
		for _, s := range h.ss {
			s.ClearIntents()
		}
		err = h.err
		h.err = nil
	}
	return
}

func (h *syncErrorHandler) OnErr(err error, _ interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	h.err = err
	return nil
}

type eventService struct {
	event istructs.IPLogEvent
}

func (s *eventService) getWSID() istructs.WSID { return s.event.Workspace() }

// implements ISchemaBuilder
type SchemaBuilder struct {
	cache                appdef.SchemaCacheBuilder
	qname                appdef.QName
	valueSchema          appdef.SchemaBuilder
	partitionKeySchema   appdef.SchemaBuilder
	clusteringColsSchema appdef.SchemaBuilder
}

func qnameValue(qname appdef.QName) appdef.QName {
	return appdef.NewQName(qname.Pkg(), qname.Entity()+"_viewValue")
}

func qnamePartitionKey(qname appdef.QName) appdef.QName {
	return appdef.NewQName(qname.Pkg(), qname.Entity()+"_viewPartitionKey")
}

func qnameClusteringCols(qname appdef.QName) appdef.QName {
	return appdef.NewQName(qname.Pkg(), qname.Entity()+"_viewClusteringCols")
}

func (me *SchemaBuilder) ValueField(name string, kind appdef.DataKind, required bool) {
	me.valueSchema.AddField(name, kind, required)
}

func (me *SchemaBuilder) PartitionKeyField(name string, kind appdef.DataKind, required bool) {
	me.partitionKeySchema.AddField(name, kind, required)
}

func (me *SchemaBuilder) ClusteringColumnField(name string, kind appdef.DataKind, required bool) {
	me.clusteringColsSchema.AddField(name, kind, required)
}

func newSchemaBuilder(cache appdef.SchemaCacheBuilder, qname appdef.QName) SchemaBuilder {
	return SchemaBuilder{
		cache:                cache,
		qname:                qname,
		valueSchema:          cache.Add(qnameValue(qname), appdef.SchemaKind_ViewRecord_Value),
		partitionKeySchema:   cache.Add(qnamePartitionKey(qname), appdef.SchemaKind_ViewRecord_PartitionKey),
		clusteringColsSchema: cache.Add(qnameClusteringCols(qname), appdef.SchemaKind_ViewRecord_ClusteringColumns),
	}
}

func provideViewSchemaImpl(cache appdef.SchemaCacheBuilder, qname appdef.QName, buildFunc BuildViewSchemaFunc) {
	builder := newSchemaBuilder(cache, qname)
	buildFunc(&builder)

	schema := cache.Add(qname, appdef.SchemaKind_ViewRecord)
	schema.AddContainer(appdef.SystemContainer_ViewPartitionKey, qnamePartitionKey(qname), 1, 1)
	schema.AddContainer(appdef.SystemContainer_ViewClusteringCols, qnameClusteringCols(qname), 1, 1)
	schema.AddContainer(appdef.SystemContainer_ViewValue, qnameValue(qname), 1, 1)
}

func provideOffsetsSchemaImpl(cache appdef.SchemaCacheBuilder) {
	offsetsSchema := cache.Add(qnameProjectionOffsets, appdef.SchemaKind_ViewRecord)
	offsetsSchema.AddContainer(appdef.SystemContainer_ViewPartitionKey, qnameProjectionOffsetsPartitionKey, 1, 1)
	offsetsSchema.AddContainer(appdef.SystemContainer_ViewClusteringCols, qnameProjectionOffsetsClusteringCols, 1, 1)
	offsetsSchema.AddContainer(appdef.SystemContainer_ViewValue, qnameProjectionOffsetsValue, 1, 1)

	partitionKeySchema := cache.Add(qnameProjectionOffsetsPartitionKey, appdef.SchemaKind_ViewRecord_PartitionKey)
	partitionKeySchema.AddField(partitionFld, appdef.DataKind_int32, true) // partitionID is uint16

	offsetsKeySchema := cache.Add(qnameProjectionOffsetsClusteringCols, appdef.SchemaKind_ViewRecord_ClusteringColumns)
	offsetsKeySchema.AddField(projectorNameFld, appdef.DataKind_QName, true)

	offsetsValueSchema := cache.Add(qnameProjectionOffsetsValue, appdef.SchemaKind_ViewRecord_Value)
	offsetsValueSchema.AddField(offsetFld, appdef.DataKind_int64, true)
}
