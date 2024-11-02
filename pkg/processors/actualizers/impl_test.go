/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	wsdescutil "github.com/voedger/voedger/pkg/coreutils/testwsdesc"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istoragecache"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/vvm/engines"
)

var newWorkspaceCmd = appdef.NewQName("sys", "NewWorkspace")

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

	appParts, appStructs, start, stop := deployTestApp(
		istructs.AppQName_test1_app1, 1, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).SetSync(true).Events().Add(testQName, appdef.ProjectorEventKind_Execute)
			wsb.AddProjector(decrementorName).SetSync(true).Events().Add(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.AddSyncProjectors(testIncrementor, testDecrementor)
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		},
		&BasicAsyncActualizerConfig{})

	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1))
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1))

	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})

	start()
	defer stop()

	t.Run("Emulate the command processor", func(t *testing.T) {
		proc := cmdProcMock{appParts}

		proc.TestEvent(1001)
		proc.TestEvent(1001)
		proc.TestEvent(1002)
		proc.TestEvent(1001)
		proc.TestEvent(1001)
		proc.TestEvent(1001)
		proc.TestEvent(1002)
		proc.TestEvent(1002)
	})

	// now read the projection values in workspaces
	require.EqualValues(5, getProjectionValue(require, appStructs, incProjectionView, 1001))
	require.EqualValues(3, getProjectionValue(require, appStructs, incProjectionView, 1002))
	require.EqualValues(-5, getProjectionValue(require, appStructs, decProjectionView, 1001))
	require.EqualValues(-3, getProjectionValue(require, appStructs, decProjectionView, 1002))
}

var (
	incrementorName = appdef.NewQName("test", "incrementor_projector")
	decrementorName = appdef.NewQName("test", "decrementor_projector")
)

var incProjectionView = appdef.NewQName("pkg", "Incremented")
var decProjectionView = appdef.NewQName("pkg", "Decremented")
var testWorkspace = appdef.NewQName("pkg", "TestWorkspace")
var testWorkspaceDescriptor = appdef.NewQName("pkg", "TestWorkspaceDescriptor")
var errTestError = errors.New("test error")

var (
	testIncrementor = istructs.Projector{
		Name: incrementorName,
		Func: func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
			wsid := event.Workspace()
			if wsid == 1099 {
				return errTestError
			}
			key, err := s.KeyBuilder(sys.Storage_View, incProjectionView)
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
		},
	}
	testDecrementor = istructs.Projector{
		Name: decrementorName,
		Func: func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
			key, err := s.KeyBuilder(sys.Storage_View, decProjectionView)
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
		},
	}
)

var buildProjectionView = func(view appdef.IViewBuilder) {
	view.Key().PartKey().AddField("pk", appdef.DataKind_int32)
	view.Key().ClustCols().AddField("cc", appdef.DataKind_int32)
	view.Value().AddField(colValue, appdef.DataKind_int32, true)
}

type (
	wsBuildCallback func(appdef.IWorkspaceBuilder)
	appCfgCallback  func(cfg *istructsmem.AppConfigType)
)

func deployTestApp(
	appName appdef.AppQName,
	appPartsCount istructs.NumAppPartitions,
	cachedStorage bool,
	wsKind, wsDescriptorKind appdef.QName,
	wsBuild wsBuildCallback,
	prepareAppCfg appCfgCallback,
	actualizerCfg *BasicAsyncActualizerConfig,
) (
	appParts appparts.IAppPartitions,
	appStructs istructs.IAppStructs,
	start, stop func(),
) {
	appParts, _, appStructs, start, stop = deployTestAppEx(
		appName,
		appPartsCount,
		cachedStorage,
		wsKind,
		wsDescriptorKind,
		wsBuild,
		prepareAppCfg,
		actualizerCfg,
	)
	return appParts, appStructs, start, stop
}

func deployTestAppEx(
	appName appdef.AppQName,
	appPartsCount istructs.NumAppPartitions,
	cachedStorage bool,
	wsKind, wsDescriptorKind appdef.QName,
	wsBuild wsBuildCallback,
	prepareAppCfg appCfgCallback,
	actualizerCfg *BasicAsyncActualizerConfig,
) (
	appParts appparts.IAppPartitions,
	actualizers IActualizersService,
	appStructs istructs.IAppStructs,
	start, stop func(),
) {
	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(wsKind)
	descr := wsb.AddCDoc(wsDescriptorKind)
	descr.AddField(authnz.Field_WSKind, appdef.DataKind_QName, true)
	descr.SetSingleton()
	wsb.SetDescriptor(wsDescriptorKind)

	wsdescutil.AddWorkspaceDescriptorStubDef(wsb)

	if wsBuild != nil {
		wsBuild(wsb)
	}
	wsb.AddCommand(newWorkspaceCmd)

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	statelessResources := istructsmem.NewStatelessResources()
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
		cfg.Resources.Add(istructsmem.NewCommandFunction(newWorkspaceCmd, istructsmem.NullCommandExec))
	}

	appDef, err := adb.Build()
	if err != nil {
		panic(err)
	}

	var vvmName string = "testVVM"

	if actualizerCfg.VvmName == "" {
		actualizerCfg.VvmName = vvmName
	} else {
		vvmName = actualizerCfg.VvmName
	}

	vvmCtx, vvmCancel := context.WithCancel(context.Background())

	var metrics imetrics.IMetrics

	if actualizerCfg.Metrics == nil {
		metrics = imetrics.Provide()
		actualizerCfg.Metrics = metrics
	} else {
		metrics = actualizerCfg.Metrics
	}

	var storageProvider istorage.IAppStorageProvider

	if cachedStorage {
		storageProvider = istoragecache.Provide(1000000, istorageimpl.Provide(mem.Provide()), metrics, vvmName)
	} else {
		storageProvider = istorageimpl.Provide(mem.Provide())
	}

	var (
		n10nBroker in10n.IN10nBroker
		n10cleanup func()
	)

	if actualizerCfg.Broker == nil {
		n10nBroker, n10cleanup = in10nmem.ProvideEx2(in10n.Quotas{
			Channels:                1000,
			ChannelsPerSubject:      10,
			Subscriptions:           1000,
			SubscriptionsPerSubject: 10,
		}, coreutils.NewITime())
		actualizerCfg.Broker = n10nBroker
	} else {
		n10nBroker = actualizerCfg.Broker
	}

	appStructsProvider := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider)

	appStructs, err = appStructsProvider.BuiltIn(appName)
	if err != nil {
		panic(err)
	}

	var secretReader isecrets.ISecretReader

	if actualizerCfg.SecretReader == nil {
		secretReader = isecretsimpl.ProvideSecretReader()
		actualizerCfg.SecretReader = secretReader
	} else {
		secretReader = actualizerCfg.SecretReader
	}

	actualizers = ProvideActualizers(*actualizerCfg)

	appParts, appPartsCleanup, err := appparts.New2(
		vvmCtx,
		appStructsProvider,
		NewSyncActualizerFactoryFactory(ProvideSyncActualizerFactory(), secretReader, n10nBroker, statelessResources),
		actualizers,
		appparts.NullSchedulerRunner, // no job schedulers
		engines.ProvideExtEngineFactories(
			engines.ExtEngineFactoriesConfig{
				AppConfigs:         cfgs,
				StatelessResources: statelessResources,
				WASMConfig:         iextengine.WASMFactoryConfig{Compile: false},
			}, "", imetrics.Provide()))
	if err != nil {
		panic(err)
	}

	appParts.DeployApp(appName, nil, appDef, appPartsCount, appparts.PoolSize(10, 10, 10, 0), cfg.NumAppWorkspaces())

	start = func() {
		if err := actualizers.Prepare(struct{}{}); err != nil {
			panic(err)
		}
		actualizers.RunEx(vvmCtx, func() {})
	}

	stop = func() {
		vvmCancel()
		actualizers.Stop()
		appPartsCleanup()
		if n10cleanup != nil {
			n10cleanup()
		}
	}

	return appParts, actualizers, appStructs, start, stop
}

func createWS(appStructs istructs.IAppStructs, ws istructs.WSID, wsKind, wsDescriptorKind appdef.QName, partition istructs.PartitionID, offset istructs.Offset) {
	now := time.Now()
	// Create workspace
	rebWs := appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         ws,
			HandlingPartition: partition,
			PLogOffset:        offset,
			QName:             newWorkspaceCmd,
		},
	})
	cud := rebWs.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
	cud.PutRecordID(appdef.SystemField_ID, 1)
	cud.PutQName(authnz.Field_WSKind, wsDescriptorKind)
	cud.PutInt32("Status", int32(authnz.WorkspaceStatus_Active))
	cud.PutInt64("InitCompletedAtMs", now.UnixMilli())
	cud.PutString(authnz.Field_WSName, wsKind.Entity())
	rawWsEvent, err := rebWs.BuildRawEvent()
	if err != nil {
		panic(err)
	}
	wsEvent, err := appStructs.Events().PutPlog(rawWsEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		panic(err)
	}
	appStructs.Records().Apply(wsEvent)
}

func Test_ErrorInSyncActualizer(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, start, stop := deployTestApp(
		istructs.AppQName_test1_app1, 1, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).SetSync(true).Events().Add(testQName, appdef.ProjectorEventKind_Execute)
			wsb.AddProjector(decrementorName).SetSync(true).Events().Add(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.AddSyncProjectors(testIncrementor, testDecrementor)
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		},
		&BasicAsyncActualizerConfig{})

	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1))
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1))
	createWS(appStructs, istructs.WSID(1099), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1))

	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})

	start()
	defer stop()

	t.Run("Emulate the command processor", func(t *testing.T) {
		proc := cmdProcMock{appParts}

		require.NoError(proc.TestEvent(1001))
		require.NoError(proc.TestEvent(1001))
		require.NoError(proc.TestEvent(1002))
		require.ErrorContains(proc.TestEvent(1099), errTestError.Error())
	})

	// now read the projection values in workspaces
	require.EqualValues(2, getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.EqualValues(1, getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1002)))
	require.EqualValues(-2, getProjectionValue(require, appStructs, decProjectionView, istructs.WSID(1001)))
	require.EqualValues(-1, getProjectionValue(require, appStructs, decProjectionView, istructs.WSID(1002)))
	require.EqualValues(0, getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1099)))
	require.EqualValues(0, getProjectionValue(require, appStructs, decProjectionView, istructs.WSID(1099)))
}
