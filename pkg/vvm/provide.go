//go:generate go run github.com/google/wire/cmd/wire
//go:build wireinject
// +build wireinject

/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/wire"
	ibus "github.com/untillpro/airs-ibus"
	router "github.com/untillpro/airs-router2"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istoragecache"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	itokensjwt "github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/collection"
	"github.com/voedger/voedger/pkg/sys/invite"
	coreutils "github.com/voedger/voedger/pkg/utils"
	dbcertcache "github.com/voedger/voedger/pkg/vvm/db_cert_cache"
	"github.com/voedger/voedger/pkg/vvm/metrics"
	"golang.org/x/crypto/acme/autocert"
)

func ProvideVVM(vvmCfg *VVMConfig, vvmIdx VVMIdxType) (voedgerVM *VoedgerVM, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	voedgerVM = &VoedgerVM{vvmCtxCancel: cancel}
	vvmCfg.addProcessorChannel(
		// command processors
		// each restaurant must go to the same cmd proc -> one single cmd processor behind the each command service channel
		iprocbusmem.ChannelGroup{
			NumChannels:       int(vvmCfg.NumCommandProcessors),
			ChannelBufferSize: int(vvmCfg.NumCommandProcessors),
		},
		ProcessorChannel_Command,
	)

	vvmCfg.addProcessorChannel(
		// query processors
		// all query processors sits on a single channel because any restaurant could be served by any query proc
		iprocbusmem.ChannelGroup{
			NumChannels:       1,
			ChannelBufferSize: 0,
		},
		ProcessorChannel_Query,
	)
	vvmCfg.Quotas = in10n.Quotas{
		Channels:               int(DefaultQuotasChannelsFactor * vvmCfg.NumCommandProcessors),
		ChannelsPerSubject:     DefaultQuotasChannelsPerSubject,
		Subsciptions:           int(DefaultQuotasSubscriptionsFactor * vvmCfg.NumCommandProcessors),
		SubsciptionsPerSubject: DefaultQuotasSubscriptionsPerSubject,
	}
	voedgerVM.VVM, voedgerVM.vvmCleanup, err = ProvideCluster(ctx, vvmCfg, vvmIdx)
	if err != nil {
		return nil, err
	}
	return voedgerVM, BuildAppWorkspaces(voedgerVM.VVM, vvmCfg)
}

func (vvm *VoedgerVM) Shutdown() {
	vvm.vvmCtxCancel()
	vvm.ServicePipeline.Close()
	vvm.vvmCleanup()
}

func (vvm *VoedgerVM) Launch() error {
	ignition := struct{}{} // value has no sense
	return vvm.ServicePipeline.SendSync(ignition)
}

// vvmCtx must be cancelled by the caller right before vvm.ServicePipeline.Close()
func ProvideCluster(vvmCtx context.Context, vvmConfig *VVMConfig, vvmIdx VVMIdxType) (*VVM, func(), error) {
	panic(wire.Build(
		wire.Struct(new(VVM), "*"),
		wire.Struct(new(apps.APIs), "*"),
		provideServicePipeline,
		provideCommandProcessors,
		provideQueryProcessors,
		provideAppServiceFactory,
		provideAppPartitionFactory,
		provideSyncActualizerFactory,
		provideAsyncActualizersFactory,
		provideRouterServiceFactory,
		provideOperatorAppServices,
		provideBlobAppStorage,
		provideBlobberAppStruct,
		provideVVMApps,
		provideBlobberClusterAppID,
		provideServiceChannelFactory,
		provideBlobStorage,
		provideChannelGroups,
		provideProcessorChannelGroupIdxCommand,
		provideProcessorChannelGroupIdxQuery,
		provideCommandProcessorsAmount,
		provideQueryChannel,
		provideCommandChannelFactory,
		provideAppConfigs,
		provideIBus,
		provideRouterParams,
		provideRouterAppStorage,
		provideIFederation,
		provideCachingAppStorageProvider,  // IAppStorageProvider
		itokensjwt.ProvideITokens,         // ITokens
		istructsmem.Provide,               // IAppStructsProvider
		payloads.ProvideIAppTokensFactory, // IAppTokensFactory
		in10nmem.ProvideEx,
		queryprocessor.ProvideServiceFactory,
		commandprocessor.ProvideServiceFactory,
		metrics.ProvideMetricsService,
		dbcertcache.ProvideDbCache,
		imetrics.Provide,
		projectors.ProvideSyncActualizerFactory,
		projectors.ProvideAsyncActualizerFactory,
		iprocbusmem.Provide,
		provideRouterServices,
		provideMetricsServiceOperator,
		provideMetricsServicePortGetter,
		provideMetricsServicePort,
		provideVVMPortSource,
		iauthnzimpl.NewDefaultAuthenticator,
		iauthnzimpl.NewDefaultAuthorizer,
		provideAppsWSAmounts,
		provideSecretKeyJWT,
		provideSecretReader,
		provideBucketsFactory,
		provideAppsExtensionPoints,
		provideSubjectGetterFunc,
		provideStorageFactory,
		provideIAppStorageUncachingProviderFactory,
		// wire.Value(vvmConfig.NumCommandProcessors) -> (wire bug?) value github.com/untillpro/airs-bp3/vvm.CommandProcessorsCount can't be used: vvmConfig is not declared in package scope
		wire.FieldsOf(&vvmConfig,
			"NumCommandProcessors",
			"NumQueryProcessors",
			"TimeFunc",
			"Quotas",
			"BlobberServiceChannels",
			"BLOBMaxSize",
			"Name",
			"MaxPrepareQueries",
			"StorageCacheSize",
			"BusTimeout",
			"VVMPort",
			"MetricsServicePort",
			"ActualizerStateOpts",
		),
	))
}

func provideIAppStorageUncachingProviderFactory(factory istorage.IAppStorageFactory) IAppStorageUncachingProviderFactory {
	return func() (provider istorage.IAppStorageProvider) {
		return istorageimpl.Provide(factory)
	}
}

func provideStorageFactory(vvmConfig *VVMConfig) (provider istorage.IAppStorageFactory, err error) {
	return vvmConfig.StorageFactory()
}

func provideSubjectGetterFunc() iauthnzimpl.SubjectGetterFunc {
	return func(requestContext context.Context, name string, as istructs.IAppStructs, wsid istructs.WSID) ([]appdef.QName, error) {
		kb := as.ViewRecords().KeyBuilder(collection.QNameViewCollection)
		kb.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
		kb.PutQName(collection.Field_DocQName, invite.QNameCDocSubject)
		res := []appdef.QName{}
		err := as.ViewRecords().Read(requestContext, wsid, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			record := value.AsRecord(collection.Field_Record)
			if record.AsString(invite.Field_Login) != name {
				return nil
			}
			roles := strings.Split(record.AsString(invite.Field_Roles), ",")
			for _, role := range roles {
				roleQName, err := appdef.ParseQName(role)
				if err != nil {
					// notest
					// must be gauranted by the side that inserted this qname
					return err
				}
				res = append(res, roleQName)
			}
			return nil
		})
		return res, err
	}
}

func provideBucketsFactory(timeFunc coreutils.TimeFunc) irates.BucketsFactoryType {
	return func() irates.IBuckets {
		return iratesce.Provide(timeFunc)
	}
}

func provideSecretReader() isecrets.ISecretReader {
	sr := isecretsimpl.ProvideSecretReader()
	if coreutils.IsTest() {
		return &testISecretReader{realSecretReader: sr}
	}
	return sr
}

func provideSecretKeyJWT(sr isecrets.ISecretReader) (itokensjwt.SecretKeyType, error) {
	return sr.ReadSecret(SecretKeyJWTName)
}

func provideAppsWSAmounts(vvmApps VVMApps, asp istructs.IAppStructsProvider) map[istructs.AppQName]istructs.AppWSAmount {
	res := map[istructs.AppQName]istructs.AppWSAmount{}
	for _, appQName := range vvmApps {
		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			panic(err)
		}
		res[appQName] = as.WSAmount()
	}
	return res
}

func provideMetricsServicePort(msp MetricsServicePortInitial, vvmIdx VVMIdxType) metrics.MetricsServicePort {
	if msp != 0 {
		return metrics.MetricsServicePort(msp) + metrics.MetricsServicePort(vvmIdx)
	}
	return metrics.MetricsServicePort(msp)
}

// VVMPort could be dynamic -> need a source to get the actual port later
// just calling RouterService.GetPort() causes wire cycle: RouterService requires IBus->VVMApps->FederatioURL->VVMPort->RouterService
// so we need something in the middle of FederationURL and RouterService: FederationURL reads VVMPortSource, RouterService writes it.
func provideVVMPortSource() *VVMPortSource {
	return &VVMPortSource{}
}

func provideMetricsServiceOperator(ms metrics.MetricsService) MetricsServiceOperator {
	return pipeline.ServiceOperator(ms)
}

// TODO: consider vvmIdx
func provideIFederation(cfg *VVMConfig, vvmPortSource *VVMPortSource) coreutils.IFederation {
	return coreutils.NewIFederation(func() *url.URL {
		if cfg.FederationURL != nil {
			return cfg.FederationURL
		}
		resultFU, err := url.Parse(LocalHost + ":" + strconv.Itoa(int(vvmPortSource.getter())))
		if err != nil {
			// notest
			panic(err)
		}
		return resultFU
	})
}

// Metrics service port could be dynamic -> need a func that will return the actual port
func provideMetricsServicePortGetter(ms metrics.MetricsService) func() metrics.MetricsServicePort {
	return func() metrics.MetricsServicePort {
		return metrics.MetricsServicePort(ms.(interface{ GetPort() int }).GetPort())
	}
}

func provideRouterParams(cfg *VVMConfig, port VVMPortType, vvmIdx VVMIdxType) router.RouterParams {
	res := router.RouterParams{
		WriteTimeout:         cfg.RouterWriteTimeout,
		ReadTimeout:          cfg.RouterReadTimeout,
		ConnectionsLimit:     cfg.RouterConnectionsLimit,
		HTTP01ChallengeHosts: cfg.RouterHTTP01ChallengeHosts,
		RouteDefault:         cfg.RouteDefault,
		Routes:               cfg.Routes,
		RoutesRewrite:        cfg.RoutesRewrite,
		RouteDomains:         cfg.RouteDomains,
		UseBP3:               true,
	}
	if port != 0 {
		res.Port = int(port) + int(vvmIdx)
	}
	return res
}

func provideAppConfigs(vvmConfig *VVMConfig) istructsmem.AppConfigsType {
	return istructsmem.AppConfigsType{}
}

func provideAppsExtensionPoints(vvmConfig *VVMConfig) map[istructs.AppQName]extensionpoints.IExtensionPoint {
	return vvmConfig.VVMAppsBuilder.PrepareAppsExtensionPoints()
}

func provideVVMApps(vvmConfig *VVMConfig, cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) VVMApps {
	return vvmConfig.VVMAppsBuilder.Build(cfgs, apis, appsEPs)
}

func provideServiceChannelFactory(vvmConfig *VVMConfig, procbus iprocbus.IProcBus) ServiceChannelFactory {
	return vvmConfig.ProvideServiceChannelFactory(procbus)
}

func provideCommandProcessorsAmount(vvmCfg *VVMConfig) CommandProcessorsAmountType {
	for _, pc := range vvmCfg.processorsChannels {
		if pc.ChannelType == ProcessorChannel_Command {
			return CommandProcessorsAmountType(pc.NumChannels)
		}
	}
	panic("no command processor channel group")
}

func provideProcessorChannelGroupIdxCommand(vvmCfg *VVMConfig) CommandProcessorsChannelGroupIdxType {
	return CommandProcessorsChannelGroupIdxType(getChannelGroupIdx(vvmCfg, ProcessorChannel_Command))
}

func provideProcessorChannelGroupIdxQuery(vvmCfg *VVMConfig) QueryProcessorsChannelGroupIdxType {
	return QueryProcessorsChannelGroupIdxType(getChannelGroupIdx(vvmCfg, ProcessorChannel_Query))
}

func getChannelGroupIdx(vvmCfg *VVMConfig, channelType ProcessorChannelType) int {
	for channelGroup, pc := range vvmCfg.processorsChannels {
		if pc.ChannelType == channelType {
			return channelGroup
		}
	}
	panic("wrong processor channel group config")
}

func provideChannelGroups(cfg *VVMConfig) (res []iprocbusmem.ChannelGroup) {
	for _, pc := range cfg.processorsChannels {
		res = append(res, pc.ChannelGroup)
	}
	return
}

func provideCachingAppStorageProvider(vvmCfg *VVMConfig, storageCacheSize StorageCacheSizeType, metrics imetrics.IMetrics,
	vvmName commandprocessor.VVMName, uncachingProivder IAppStorageUncachingProviderFactory) (istorage.IAppStorageProvider, error) {
	aspNonCaching := uncachingProivder()
	res := istoragecache.Provide(int(storageCacheSize), aspNonCaching, metrics, string(vvmName))
	return res, nil
}

// синхронный актуализатор один на приложение из-за storages, которые у каждого приложения свои
// сделаем так, чтобы в командный процессор подавался свитч по appName, который выберет нужный актуализатор с нужным набором проекторов
type switchByAppName struct {
}

func (s *switchByAppName) Switch(work interface{}) (branchName string, err error) {
	return work.(interface{ AppQName() istructs.AppQName }).AppQName().String(), nil
}

func provideSyncActualizerFactory(vvmApps VVMApps, structsProvider istructs.IAppStructsProvider, n10nBroker in10n.IN10nBroker, mpq MaxPrepareQueriesType, actualizerFactory projectors.SyncActualizerFactory, secretReader isecrets.ISecretReader) commandprocessor.SyncActualizerFactory {
	return func(vvmCtx context.Context, partitionID istructs.PartitionID) pipeline.ISyncOperator {
		actualizers := []pipeline.SwitchOperatorOptionFunc{}
		for _, appQName := range vvmApps {
			appStructs, err := structsProvider.AppStructs(appQName)
			if err != nil {
				panic(err)
			}
			if len(appStructs.SyncProjectors()) == 0 {
				actualizers = append(actualizers, pipeline.SwitchBranch(appQName.String(), &pipeline.NOOP{}))
				continue
			}
			conf := projectors.SyncActualizerConf{
				Ctx: vvmCtx,
				//TODO это правильно, что постоянную appStrcuts возвращаем? Каждый раз не надо запрашивать у appStructsProvider?
				AppStructs:   func() istructs.IAppStructs { return appStructs },
				SecretReader: secretReader,
				Partition:    partitionID,
				WorkToEvent: func(work interface{}) istructs.IPLogEvent {
					return work.(interface{ Event() istructs.IPLogEvent }).Event()
				},
				N10nFunc: func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {
					n10nBroker.Update(in10n.ProjectionKey{
						App:        appStructs.AppQName(),
						Projection: view,
						WS:         wsid,
					}, offset)
				},
				IntentsLimit: actualizerIntentsLimit,
			}
			actualizer := actualizerFactory(conf, appStructs.SyncProjectors()[0], appStructs.SyncProjectors()[1:]...)
			actualizers = append(actualizers, pipeline.SwitchBranch(appQName.String(), actualizer))
		}
		return pipeline.SwitchOperator(&switchByAppName{}, actualizers[0], actualizers[1:]...)
	}
}

func provideBlobberAppStruct(asp istructs.IAppStructsProvider) (BlobberAppStruct, error) {
	return asp.AppStructs(istructs.AppQName_sys_blobber)
}

func provideBlobberClusterAppID(bas BlobberAppStruct) BlobberAppClusterID {
	return BlobberAppClusterID(bas.ClusterAppID())
}

func provideBlobAppStorage(astp istorage.IAppStorageProvider) (BlobAppStorage, error) {
	return astp.AppStorage(istructs.AppQName_sys_blobber)
}

func provideBlobStorage(bas BlobAppStorage, nowFunc coreutils.TimeFunc) BlobStorage {
	return iblobstoragestg.Provide(bas, nowFunc)
}

func provideRouterAppStorage(astp istorage.IAppStorageProvider) (dbcertcache.RouterAppStorage, error) {
	return astp.AppStorage(istructs.AppQName_sys_router)
}

// port 80 -> [0] is http server, port 443 -> [0] is https server, [1] is acme server
func provideRouterServices(vvmCtx context.Context, rp router.RouterParams, busTimeout BusTimeout, broker in10n.IN10nBroker, quotas in10n.Quotas,
	nowFunc coreutils.TimeFunc, bsc router.BlobberServiceChannels, bms router.BLOBMaxSizeType, blobberClusterAppID BlobberAppClusterID, blobStorage BlobStorage,
	routerAppStorage dbcertcache.RouterAppStorage, autocertCache autocert.Cache, bus ibus.IBus, vvmPortSource *VVMPortSource, appsWSAmounts map[istructs.AppQName]istructs.AppWSAmount) RouterServices {
	bp := &router.BlobberParams{
		ClusterAppBlobberID:    uint32(blobberClusterAppID),
		ServiceChannels:        bsc,
		BLOBStorage:            blobStorage,
		BLOBWorkersNum:         DefaultBLOBWorkersNum,
		RetryAfterSecondsOn503: DefaultRetryAfterSecondsOn503,
		BLOBMaxSize:            bms,
	}
	res := router.ProvideBP3(vvmCtx, rp, time.Duration(busTimeout), broker, quotas, bp, autocertCache, bus, appsWSAmounts)
	vvmPortSource.getter = func() VVMPortType {
		return VVMPortType(res[0].(interface{ GetPort() int }).GetPort())
	}
	return res
}

func provideRouterServiceFactory(rs RouterServices) RouterServiceOperator {
	routerServices := []pipeline.ForkOperatorOptionFunc{}
	for _, routerSrvIntf := range rs {
		routerServices = append(routerServices, pipeline.ForkBranch(pipeline.ServiceOperator(routerSrvIntf.(pipeline.IService))))
	}
	return pipeline.ForkOperator(pipeline.ForkSame, routerServices[0], routerServices[1:]...)
}

func provideQueryChannel(sch ServiceChannelFactory) QueryChannel {
	return QueryChannel(sch(ProcessorChannel_Query, 0))
}

func provideCommandChannelFactory(sch ServiceChannelFactory) CommandChannelFactory {
	return func(channelIdx int) commandprocessor.CommandChannel {
		return commandprocessor.CommandChannel(sch(ProcessorChannel_Command, channelIdx))
	}
}

func provideQueryProcessors(qpCount QueryProcessorsCount, qc QueryChannel, bus ibus.IBus, asp istructs.IAppStructsProvider, qpFactory queryprocessor.ServiceFactory,
	imetrics imetrics.IMetrics, vvm commandprocessor.VVMName, mpq MaxPrepareQueriesType, authn iauthnz.IAuthenticator, authz iauthnz.IAuthorizer,
	appCfgs istructsmem.AppConfigsType) OperatorQueryProcessors {
	forks := make([]pipeline.ForkOperatorOptionFunc, qpCount)
	resultSenderFactory := func(ctx context.Context, sender interface{}) queryprocessor.IResultSenderClosable {
		return &resultSenderErrorFirst{
			ctx:    ctx,
			sender: sender,
			bus:    bus,
		}
	}
	for i := 0; i < int(qpCount); i++ {
		forks[i] = pipeline.ForkBranch(pipeline.ServiceOperator(qpFactory(iprocbus.ServiceChannel(qc), resultSenderFactory, asp, int(mpq), imetrics,
			string(vvm), authn, authz, appCfgs)))
	}
	return pipeline.ForkOperator(pipeline.ForkSame, forks[0], forks[1:]...)
}

func provideCommandProcessors(cpCount coreutils.CommandProcessorsCount, ccf CommandChannelFactory, cpFactory commandprocessor.ServiceFactory) OperatorCommandProcessors {
	forks := make([]pipeline.ForkOperatorOptionFunc, cpCount)
	for i := 0; i < int(cpCount); i++ {
		forks[i] = pipeline.ForkBranch(pipeline.ServiceOperator(cpFactory(ccf(i), istructs.PartitionID(i))))
	}
	return pipeline.ForkOperator(pipeline.ForkSame, forks[0], forks[1:]...)
}

func provideAsyncActualizersFactory(appStructsProvider istructs.IAppStructsProvider, n10nBroker in10n.IN10nBroker, asyncActualizerFactory projectors.AsyncActualizerFactory, secretReader isecrets.ISecretReader) AsyncActualizersFactory {
	return func(vvmCtx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories, partitionID istructs.PartitionID, opts []state.ActualizerStateOptFunc) pipeline.ISyncOperator {
		var asyncProjectors []pipeline.ForkOperatorOptionFunc
		appStructs, err := appStructsProvider.AppStructs(appQName)
		if err != nil {
			panic(err)
		}

		conf := projectors.AsyncActualizerConf{
			Ctx:      vvmCtx,
			AppQName: appQName,
			// FIXME: это правильно, что постоянную appStrcuts возвращаем? Каждый раз не надо запрашивать у appStructsProvider?
			AppStructs:   func() istructs.IAppStructs { return appStructs },
			SecretReader: secretReader,
			Partition:    partitionID,
			Broker:       n10nBroker,
			Opts:         opts,
			IntentsLimit: actualizerIntentsLimit,
		}

		asyncProjectors = make([]pipeline.ForkOperatorOptionFunc, len(asyncProjectorFactories))

		for i, asyncProjectorFactory := range asyncProjectorFactories {
			asyncProjector, err := asyncActualizerFactory(conf, asyncProjectorFactory)
			if err != nil {
				panic(err)
			}
			asyncProjectors[i] = pipeline.ForkBranch(asyncProjector)
		}
		return pipeline.ForkOperator(func(work interface{}, branchNumber int) (fork interface{}, err error) { return struct{}{}, nil }, asyncProjectors[0], asyncProjectors[1:]...)
	}
}

func provideAppPartitionFactory(aaf AsyncActualizersFactory, opts []state.ActualizerStateOptFunc) AppPartitionFactory {
	return func(vvmCtx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories, partitionID istructs.PartitionID) pipeline.ISyncOperator {
		return aaf(vvmCtx, appQName, asyncProjectorFactories, partitionID, opts)
	}
}

func provideAppServiceFactory(apf AppPartitionFactory, cpCount coreutils.CommandProcessorsCount) AppServiceFactory {
	return func(vvmCtx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories) pipeline.ISyncOperator {
		forks := make([]pipeline.ForkOperatorOptionFunc, cpCount)
		for i := 0; i < int(cpCount); i++ {
			forks[i] = pipeline.ForkBranch(apf(vvmCtx, appQName, asyncProjectorFactories, istructs.PartitionID(i)))
		}
		return pipeline.ForkOperator(pipeline.ForkSame, forks[0], forks[1:]...)
	}
}

func provideOperatorAppServices(apf AppServiceFactory, vvmApps VVMApps, asp istructs.IAppStructsProvider) OperatorAppServicesFactory {
	return func(vvmCtx context.Context) pipeline.ISyncOperator {
		var branches []pipeline.ForkOperatorOptionFunc
		for _, appQName := range vvmApps {
			as, err := asp.AppStructs(appQName)
			if err != nil {
				panic(err)
			}
			if len(as.AsyncProjectors()) == 0 {
				continue
			}
			branch := pipeline.ForkBranch(apf(vvmCtx, appQName, as.AsyncProjectors()))
			branches = append(branches, branch)
		}
		if len(branches) == 0 {
			return &pipeline.NOOP{}
		}
		return pipeline.ForkOperator(pipeline.ForkSame, branches[0], branches[1:]...)
	}
}

func provideServicePipeline(vvmCtx context.Context, opCommandProcessors OperatorCommandProcessors, opQueryProcessors OperatorQueryProcessors, opAppServices OperatorAppServicesFactory,
	routerServiceOp RouterServiceOperator, metricsServiceOp MetricsServiceOperator) ServicePipeline {
	return pipeline.NewSyncPipeline(vvmCtx, "ServicePipeline",
		pipeline.WireSyncOperator("service fork operator", pipeline.ForkOperator(pipeline.ForkSame,

			// VVM
			pipeline.ForkBranch(pipeline.ForkOperator(pipeline.ForkSame,
				pipeline.ForkBranch(opQueryProcessors),
				pipeline.ForkBranch(opCommandProcessors),
				pipeline.ForkBranch(opAppServices(vvmCtx)), // vvmCtx here is for AsyncActualizerConf at AsyncActualizerFactory only
			)),

			// Router
			// vvmCtx here is for blobber service to stop reading from ServiceChannel on VVM shutdown
			pipeline.ForkBranch(routerServiceOp),

			// Metrics http service
			pipeline.ForkBranch(metricsServiceOp),
		)),
	)
}
