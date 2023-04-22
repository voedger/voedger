/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/schemas"
	"github.com/voedger/voedger/pkg/state"
)

// Design: Projection Actualizers
// https://dev.heeus.io/launchpad/#!12850
//
// Test description:
//
// 1. Creates sync actualizer initialized with two
// projectors: incrementor, decrementor
// (increments/decrements counter for the event's workspace)
//
// 2. Creates command processor pipeline with
// sync actualizer operator
//
// 3. Feeds command processor with events for
// different workspaces
//
// 4. The projection values for those workspaces checked
func TestBasicUsage_SynchronousActualizer(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(schemas schemas.SchemaCacheBuilder) {
			ProvideViewSchema(schemas, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(schemas, decProjectionView, buildProjectionSchema)
		},
		nil)
	actualizerFactory := ProvideSyncActualizerFactory()

	// create actualizer with two factories
	conf := SyncActualizerConf{
		Ctx:        context.Background(),
		Partition:  istructs.PartitionID(1),
		AppStructs: func() istructs.IAppStructs { return app },
	}
	actualizer := actualizerFactory(conf, incrementorFactory, decrementorFactory)

	// create partition processor pipeline
	processor := pipeline.NewSyncPipeline(context.Background(), "partition processor", pipeline.WireSyncOperator("actualizer", actualizer))

	// feed partition processor
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1002}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1002}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1002}))

	// now read the projection values in workspaces
	require.Equal(int32(5), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(3), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
	require.Equal(int32(-5), getProjectionValue(require, app, decProjectionView, istructs.WSID(1001)))
	require.Equal(int32(-3), getProjectionValue(require, app, decProjectionView, istructs.WSID(1002)))
}

var (
	incrementorName = schemas.NewQName("test", "incremenor_projector")
	decrementorName = schemas.NewQName("test", "decrementor_projector")
)

var incProjectionView = schemas.NewQName("pkg", "Incremented")
var decProjectionView = schemas.NewQName("pkg", "Decremented")

var (
	incrementorFactory = func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{Name: incrementorName, Func: incrementor}
	}
	decrementorFactory = func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{Name: decrementorName, Func: decrementor}
	}
)

var (
	incrementor = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		wsid := event.Workspace()
		if wsid == 1099 {
			return errors.New("test err")
		}
		key, err := s.KeyBuilder(state.ViewRecordsStorage, incProjectionView)
		if err != nil {
			return
		}
		key.PutInt32("pk", 0)
		key.PutInt32("cc", 0)
		el, ok, err := s.CanExist(key)
		if err != nil {
			return
		}
		eb, err := intents.NewValue(key)
		if err != nil {
			return
		}
		if ok {
			eb.PutInt32("myvalue", el.AsInt32("myvalue")+1)
		} else {
			eb.PutInt32("myvalue", 1)
		}
		return
	}
	decrementor = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		key, err := s.KeyBuilder(state.ViewRecordsStorage, decProjectionView)
		if err != nil {
			return
		}
		key.PutInt32("pk", 0)
		key.PutInt32("cc", 0)
		el, ok, err := s.CanExist(key)
		if err != nil {
			return
		}
		eb, err := intents.NewValue(key)
		if err != nil {
			return
		}
		if ok {
			eb.PutInt32("myvalue", el.AsInt32("myvalue")-1)
		} else {
			eb.PutInt32("myvalue", -1)
		}
		return
	}
)

var buildProjectionSchema = func(builder IViewSchemaBuilder) {
	builder.PartitionKeyField("pk", schemas.DataKind_int32, false)
	builder.ClusteringColumnField("cc", schemas.DataKind_int32, false)
	builder.ValueField(colValue, schemas.DataKind_int32, true)
}

type (
	schemasCfgCallback func(schemas schemas.SchemaCacheBuilder)
	appCfgCallback     func(cfg *istructsmem.AppConfigType)
)

func appStructs(schemasCfg schemasCfgCallback, appCfg appCfgCallback) istructs.IAppStructs {
	cache := schemas.NewSchemaCache()
	cache.Add(incrementorName, schemas.SchemaKind_Object)
	cache.Add(decrementorName, schemas.SchemaKind_Object)
	if schemasCfg != nil {
		schemasCfg(cache)
	}

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, cache)
	if appCfg != nil {
		appCfg(cfg)
	}

	asf := istorage.ProvideMem()
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider)
	structs, err := prov.AppStructs(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	return structs
}

func Test_ErrorInSyncActualizer(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(schemas schemas.SchemaCacheBuilder) {
			ProvideViewSchema(schemas, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(schemas, decProjectionView, buildProjectionSchema)
		},
		nil)
	actualizerFactory := ProvideSyncActualizerFactory()

	// create actualizer with two factories
	conf := SyncActualizerConf{
		Ctx:        context.Background(),
		Partition:  istructs.PartitionID(1),
		AppStructs: func() istructs.IAppStructs { return app },
	}
	actualizer := actualizerFactory(conf, incrementorFactory, decrementorFactory)

	// create partition processor pipeline
	processor := pipeline.NewSyncPipeline(context.Background(), "partition processor", pipeline.WireSyncOperator("actualizer", actualizer))

	// feed partition processor
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1001}))
	require.NoError(processor.SendSync(&plogEvent{wsid: 1002}))
	err := processor.SendSync(&plogEvent{wsid: 1099})
	require.NotNil(err)
	require.Equal("[actualizer/doSync] [ErrorHandler/doSync] [SyncActualizer/doSync] [Projector/doSync] test err", err.Error())

	// now read the projection values in workspaces
	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(1), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
	require.Equal(int32(-2), getProjectionValue(require, app, decProjectionView, istructs.WSID(1001)))
	require.Equal(int32(-1), getProjectionValue(require, app, decProjectionView, istructs.WSID(1002)))
	require.Equal(int32(0), getProjectionValue(require, app, incProjectionView, istructs.WSID(1099)))
	require.Equal(int32(0), getProjectionValue(require, app, decProjectionView, istructs.WSID(1099)))
}
