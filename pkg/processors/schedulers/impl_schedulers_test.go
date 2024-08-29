/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

import (
	"context"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
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
	"github.com/voedger/voedger/pkg/processors/actualizers"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/vvm/engines"
)

var incProjectionView = appdef.NewQName("pkg", "Incremented")
var testWorkspace = appdef.NewQName("pkg", "TestWorkspace")
var testWorkspaceDescriptor = appdef.NewQName("pkg", "TestWorkspaceDescriptor")
var jobName = appdef.NewQName("pkg", "IncrementJob")

func TestSchedulersService(t *testing.T) {
	config := &BasicSchedulerConfig{
		VvmName: "test",
	}

	appParts, schedulers, appStructs, start, stop := deployTestApp(
		istructs.AppQName_test1_app1, 1, false,
		func(appDef appdef.IAppDefBuilder) {
			appDef.AddPackage("test", "test.com/test")
			actualizers.ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			appDef.AddJob(jobName).SetCronSchedule("* * * * * *").SetEngine(appdef.ExtensionEngineKind_BuiltIn?)
			ws := addWS(appDef, testWorkspace, testWorkspaceDescriptor)
			ws.AddType(incProjectionView)
		},
		func(cfg *istructsmem.AppConfigType) {
		},
		config)

	createWS(appStructs, istructs.WSID(1001), testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1))

	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})

	start()
	defer stop()

	go func() { schedulers.NewAndRun(context.Background(), istructs.AppQName_test1_app1, 1, 0, 1001, jobName) }()

}

type (
	appDefCallback func(appDef appdef.IAppDefBuilder)
	appCfgCallback func(cfg *istructsmem.AppConfigType)
)

var newWorkspaceCmd = appdef.NewQName("sys", "NewWorkspace")

var buildProjectionView = func(view appdef.IViewBuilder) {
	view.Key().PartKey().AddField("pk", appdef.DataKind_int32)
	view.Key().ClustCols().AddField("cc", appdef.DataKind_int32)
	view.Value().AddField("val", appdef.DataKind_int32, true)
}

func createWS(appStructs istructs.IAppStructs, ws istructs.WSID, wsDescriptorKind appdef.QName, partition istructs.PartitionID, offset istructs.Offset) {
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
	cud.PutQName("WSKind", wsDescriptorKind)
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

func addWS(appDef appdef.IAppDefBuilder, wsKind, wsDescriptorKind appdef.QName) appdef.IWorkspaceBuilder {
	descr := appDef.AddCDoc(wsDescriptorKind)
	descr.AddField("WSKind", appdef.DataKind_QName, true)
	ws := appDef.AddWorkspace(wsKind)
	ws.SetDescriptor(wsDescriptorKind)
	return ws
}

func deployTestApp(
	appName appdef.AppQName,
	appPartsCount istructs.NumAppPartitions,
	cachedStorage bool,
	prepareAppDef appDefCallback,
	prepareAppCfg appCfgCallback,
	schedulersCfg *BasicSchedulerConfig,
) (
	appParts appparts.IAppPartitions,
	schedulers ISchedulersService,
	appStructs istructs.IAppStructs,
	start, stop func(),
) {
	appDefBuilder := appdef.New()
	if prepareAppDef != nil {
		prepareAppDef(appDefBuilder)
	}
	appDefBuilder.AddCommand(newWorkspaceCmd)

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, appDefBuilder)
	statelessResources := istructsmem.NewStatelessResources()
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
		cfg.Resources.Add(istructsmem.NewCommandFunction(newWorkspaceCmd, istructsmem.NullCommandExec))
	}

	wsDescr := appDefBuilder.AddCDoc(authnz.QNameCDocWorkspaceDescriptor)
	wsDescr.AddField(authnz.Field_WSKind, appdef.DataKind_QName, true)
	wsDescr.SetSingleton()

	appDef, err := appDefBuilder.Build()
	if err != nil {
		panic(err)
	}

	var vvmName string = "testVVM"

	if schedulersCfg.VvmName == "" {
		schedulersCfg.VvmName = vvmName
	} else {
		vvmName = schedulersCfg.VvmName
	}

	vvmCtx, vvmCancel := context.WithCancel(context.Background())

	var metrics imetrics.IMetrics

	if schedulersCfg.Metrics == nil {
		metrics = imetrics.Provide()
		schedulersCfg.Metrics = metrics
	} else {
		metrics = schedulersCfg.Metrics
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

	if schedulersCfg.Broker == nil {
		n10nBroker, n10cleanup = in10nmem.ProvideEx2(in10n.Quotas{
			Channels:                1000,
			ChannelsPerSubject:      10,
			Subscriptions:           1000,
			SubscriptionsPerSubject: 10,
		}, time.Now)
		schedulersCfg.Broker = n10nBroker
	} else {
		n10nBroker = schedulersCfg.Broker
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

	if schedulersCfg.SecretReader == nil {
		secretReader = isecretsimpl.ProvideSecretReader()
		schedulersCfg.SecretReader = secretReader
	} else {
		secretReader = schedulersCfg.SecretReader
	}

	schedulers = ProvideSchedulers(*schedulersCfg)

	appParts, appPartsCleanup, err := appparts.New2(
		vvmCtx,
		appStructsProvider,
		appparts.NullSyncActualizerFactory,
		appparts.NullActualizerRunner,
		schedulers,
		engines.ProvideExtEngineFactories(
			engines.ExtEngineFactoriesConfig{
				AppConfigs:         cfgs,
				StatelessResources: statelessResources,
				WASMConfig:         iextengine.WASMFactoryConfig{Compile: false},
			}))
	if err != nil {
		panic(err)
	}

	appParts.DeployApp(appName, nil, appDef, appPartsCount, appparts.PoolSize(10, 10, 10, 0), cfg.NumAppWorkspaces())

	start = func() {
		if err := schedulers.Prepare(struct{}{}); err != nil {
			panic(err)
		}
		schedulers.RunEx(vvmCtx, func() {})
	}

	stop = func() {
		vvmCancel()
		schedulers.Stop()
		appPartsCleanup()
		if n10cleanup != nil {
			n10cleanup()
		}
	}

	return appParts, schedulers, appStructs, start, stop
}
