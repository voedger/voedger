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
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istoragecache"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
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

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(cmdQName)
			appDef.AddProjector(incrementorName).AddEvent(cmdQName, appdef.ProjectorEventKind_Execute)
			appDef.AddProjector(decrementorName).AddEvent(cmdQName, appdef.ProjectorEventKind_Execute)
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
	incrementorName = appdef.NewQName("test", "incremenor_projector")
	decrementorName = appdef.NewQName("test", "decrementor_projector")
)

var incProjectionView = appdef.NewQName("pkg", "Incremented")
var decProjectionView = appdef.NewQName("pkg", "Decremented")

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
		key, err := s.KeyBuilder(state.View, incProjectionView)
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
		key, err := s.KeyBuilder(state.View, decProjectionView)
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

var buildProjectionView = func(view appdef.IViewBuilder) {
	view.KeyBuilder().PartKeyBuilder().AddField("pk", appdef.DataKind_int32)
	view.KeyBuilder().ClustColsBuilder().AddField("cc", appdef.DataKind_int32)
	view.ValueBuilder().AddField(colValue, appdef.DataKind_int32, true)
}

type (
	appDefCallback func(appDef appdef.IAppDefBuilder)
	appCfgCallback func(cfg *istructsmem.AppConfigType)
)

func appStructs(prepareAppDef appDefCallback, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	appDef := appdef.New()
	// appDef.AddObject(incrementorName)
	// appDef.AddObject(decrementorName)
	if prepareAppDef != nil {
		prepareAppDef(appDef)
	}

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
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

var metrics imetrics.IMetrics

func appStructsCached(prepareAppDef appDefCallback, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	appDef := appdef.New()
	appDef.AddObject(incrementorName)
	appDef.AddObject(decrementorName)
	if prepareAppDef != nil {
		prepareAppDef(appDef)
	}

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
	}

	asf := istorage.ProvideMem()
	metrics = imetrics.Provide()
	storageProvider := istorageimpl.Provide(asf)
	cached := istoragecache.Provide(1000000, storageProvider, metrics, "testVM")
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		cached)
	structs, err := prov.AppStructs(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	return structs
}

func Test_ErrorInSyncActualizer(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
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
	require.Equal("test err", err.Error())

	// now read the projection values in workspaces
	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(1), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
	require.Equal(int32(-2), getProjectionValue(require, app, decProjectionView, istructs.WSID(1001)))
	require.Equal(int32(-1), getProjectionValue(require, app, decProjectionView, istructs.WSID(1002)))
	require.Equal(int32(0), getProjectionValue(require, app, incProjectionView, istructs.WSID(1099)))
	require.Equal(int32(0), getProjectionValue(require, app, decProjectionView, istructs.WSID(1099)))
}
