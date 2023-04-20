/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"fmt"

	istructs "github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/schemas"
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
	schemas              schemas.SchemaCacheBuilder
	qname                istructs.QName
	valueSchema          schemas.SchemaBuilder
	partitionKeySchema   schemas.SchemaBuilder
	clusteringColsSchema schemas.SchemaBuilder
}

func qnameValue(qname istructs.QName) istructs.QName {
	return istructs.NewQName(qname.Pkg(), qname.Entity()+"_viewValue")
}

func qnamePartitionKey(qname istructs.QName) istructs.QName {
	return istructs.NewQName(qname.Pkg(), qname.Entity()+"_viewPartitionKey")
}

func qnameClusteringCols(qname istructs.QName) istructs.QName {
	return istructs.NewQName(qname.Pkg(), qname.Entity()+"_viewClusteringCols")
}

func (me *SchemaBuilder) ValueField(name string, kind istructs.DataKindType, required bool) {
	me.valueSchema.AddField(name, kind, required)
}

func (me *SchemaBuilder) PartitionKeyField(name string, kind istructs.DataKindType, required bool) {
	me.partitionKeySchema.AddField(name, kind, required)
}

func (me *SchemaBuilder) ClusteringColumnField(name string, kind istructs.DataKindType, required bool) {
	me.clusteringColsSchema.AddField(name, kind, required)
}

func newSchemaBuilder(schemas schemas.SchemaCacheBuilder, qname istructs.QName) SchemaBuilder {
	return SchemaBuilder{
		schemas:              schemas,
		qname:                qname,
		valueSchema:          schemas.Add(qnameValue(qname), istructs.SchemaKind_ViewRecord_Value),
		partitionKeySchema:   schemas.Add(qnamePartitionKey(qname), istructs.SchemaKind_ViewRecord_PartitionKey),
		clusteringColsSchema: schemas.Add(qnameClusteringCols(qname), istructs.SchemaKind_ViewRecord_ClusteringColumns),
	}
}

func provideViewSchemaImpl(schemas schemas.SchemaCacheBuilder, qname istructs.QName, buildFunc BuildViewSchemaFunc) {
	builder := newSchemaBuilder(schemas, qname)
	buildFunc(&builder)

	schema := schemas.Add(qname, istructs.SchemaKind_ViewRecord)
	schema.AddContainer(istructs.SystemContainer_ViewPartitionKey, qnamePartitionKey(qname), 1, 1)
	schema.AddContainer(istructs.SystemContainer_ViewClusteringCols, qnameClusteringCols(qname), 1, 1)
	schema.AddContainer(istructs.SystemContainer_ViewValue, qnameValue(qname), 1, 1)
}

func provideOffsetsSchemaImpl(schemas schemas.SchemaCacheBuilder) {
	offsetsSchema := schemas.Add(qnameProjectionOffsets, istructs.SchemaKind_ViewRecord)
	offsetsSchema.AddContainer(istructs.SystemContainer_ViewPartitionKey, qnameProjectionOffsetsPartitionKey, 1, 1)
	offsetsSchema.AddContainer(istructs.SystemContainer_ViewClusteringCols, qnameProjectionOffsetsClusteringCols, 1, 1)
	offsetsSchema.AddContainer(istructs.SystemContainer_ViewValue, qnameProjectionOffsetsValue, 1, 1)

	partitionKeySchema := schemas.Add(qnameProjectionOffsetsPartitionKey, istructs.SchemaKind_ViewRecord_PartitionKey)
	partitionKeySchema.AddField(partitionFld, istructs.DataKind_int32, true) // partitionID is uint16

	offsetsKeySchema := schemas.Add(qnameProjectionOffsetsClusteringCols, istructs.SchemaKind_ViewRecord_ClusteringColumns)
	offsetsKeySchema.AddField(projectorNameFld, istructs.DataKind_QName, true)

	offsetsValueSchema := schemas.Add(qnameProjectionOffsetsValue, istructs.SchemaKind_ViewRecord_Value)
	offsetsValueSchema.AddField(offsetFld, istructs.DataKind_int64, true)
}
