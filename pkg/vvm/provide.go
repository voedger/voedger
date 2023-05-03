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

func ProvideHVM(hvmCfg *HVMConfig, hvmIdx HVMIdxType) (heeusVM *HeeusVM, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	heeusVM = &HeeusVM{hvmCtxCancel: cancel}
	heeusVM.HVM, heeusVM.hvmCleanup, err = ProvideCluster(ctx, hvmCfg, hvmIdx)
	if err != nil {
		return nil, err
	}
	return heeusVM, BuildAppWorkspaces(heeusVM.HVM, hvmCfg)
}

func (vvm *HeeusVM) Shutdown() {
	vvm.hvmCtxCancel()
	vvm.ServicePipeline.Close()
	vvm.hvmCleanup()
}

func (vvm *HeeusVM) Launch() error {
	ignition := struct{}{} // value has no sense
	return vvm.ServicePipeline.SendSync(ignition)
}

// hvmCtx must be cancelled by the caller right before vvm.ServicePipeline.Close()
func ProvideCluster(hvmCtx context.Context, hvmConfig *HVMConfig, hvmIdx HVMIdxType) (*HVM, func(), error) {
	panic(wire.Build(
		wire.Struct(new(HVM), "*"),
		wire.Struct(new(HVMAPI), "*"),
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
		provideHVMApps,
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
		provideFederationURL,
		provideCachingAppStorageProvider,  // IAppStorageProvider
		itokensjwt.ProvideITokens,         // ITokens
		istructsmem.Provide,               // IAppStructsProvider
		payloads.ProvideIAppTokensFactory, // IAppTokensFactory
		// istorageimpl.Provide,              // IAppstorageProvider
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
		provideHVMPortSource,
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
		// wire.Value(hvmConfig.NumCommandProcessors) -> (wire bug?) value github.com/untillpro/airs-bp3/vvm.CommandProcessorsCount can't be used: hvmConfig is not declared in package scope
		wire.FieldsOf(&hvmConfig,
			"NumCommandProcessors",
			"NumQueryProcessors",
			"PartitionsCount",
			"TimeFunc",
			"Quotas",
			"BlobberServiceChannels",
			"BLOBMaxSize",
			"Name",
			"MaxPrepareQueries",
			"StorageCacheSize",
			"BusTimeout",
			"HVMPort",
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

func provideStorageFactory(hvmConfig *HVMConfig) (provider istorage.IAppStorageFactory, err error) {
	return hvmConfig.StorageFactory()
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

func provideBucketsFactory(timeFunc func() time.Time) irates.BucketsFactoryType {
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

func provideAppsWSAmounts(hvmApps HVMApps, asp istructs.IAppStructsProvider) map[istructs.AppQName]istructs.AppWSAmount {
	res := map[istructs.AppQName]istructs.AppWSAmount{}
	for _, appQName := range hvmApps {
		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			panic(err)
		}
		res[appQName] = as.WSAmount()
	}
	return res
}

func provideMetricsServicePort(msp MetricsServicePortInitial, hvmIdx HVMIdxType) metrics.MetricsServicePort {
	if msp != 0 {
		return metrics.MetricsServicePort(msp) + metrics.MetricsServicePort(hvmIdx)
	}
	return metrics.MetricsServicePort(msp)
}

// HVMPort could be dynamic -> need a source to get the actual port later
// just calling RouterService.GetPort() causes wire cycle: RouterService requires IBus->HVMApps->FederatioURL->HVMPort->RouterService
// so we need something in the middle of FederationURL and RouterService: FederationURL reads HVMPortSource, RouterService writes it.
func provideHVMPortSource() *HVMPortSource {
	return &HVMPortSource{}
}

func provideMetricsServiceOperator(ms metrics.MetricsService) MetricsServiceOperator {
	return pipeline.ServiceOperator(ms)
}

// TODO: consider hvmIdx
func provideFederationURL(cfg *HVMConfig, hvmPortSource *HVMPortSource) FederationURLType {
	return func() *url.URL {
		if cfg.FederationURL != nil {
			return cfg.FederationURL
		}
		resultFU, err := url.Parse(LocalHost + ":" + strconv.Itoa(int(hvmPortSource.getter())))
		if err != nil {
			// notest
			panic(err)
		}
		return resultFU
	}
}

// Metrics service port could be dynamic -> need a func that will return the actual port
func provideMetricsServicePortGetter(ms metrics.MetricsService) func() metrics.MetricsServicePort {
	return func() metrics.MetricsServicePort {
		return metrics.MetricsServicePort(ms.(interface{ GetPort() int }).GetPort())
	}
}

func provideRouterParams(cfg *HVMConfig, port HVMPortType, hvmIdx HVMIdxType) router.RouterParams {
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
		res.Port = int(port) + int(hvmIdx)
	}
	return res
}

func provideAppConfigs(hvmConfig *HVMConfig) istructsmem.AppConfigsType {
	return istructsmem.AppConfigsType{}
}

func provideAppsExtensionPoints(hvmConfig *HVMConfig) map[istructs.AppQName]IStandardExtensionPoints {
	return hvmConfig.HVMAppsBuilder.PrepareStandardExtensionPoints()
}

func provideHVMApps(hvmConfig *HVMConfig, cfgs istructsmem.AppConfigsType, hvmAPI HVMAPI, seps map[istructs.AppQName]IStandardExtensionPoints) HVMApps {
	return hvmConfig.HVMAppsBuilder.Build(hvmConfig, cfgs, hvmAPI, seps)
}

func provideServiceChannelFactory(hvmConfig *HVMConfig, procbus iprocbus.IProcBus) ServiceChannelFactory {
	return hvmConfig.ProvideServiceChannelFactory(procbus)
}

func provideCommandProcessorsAmount(hvmCfg *HVMConfig) CommandProcessorsAmountType {
	for _, pc := range hvmCfg.processorsChannels {
		if pc.ChannelType == ProcessorChannel_Command {
			return CommandProcessorsAmountType(pc.NumChannels)
		}
	}
	panic("no command processor channel group")
}

func provideProcessorChannelGroupIdxCommand(hvmCfg *HVMConfig) CommandProcessorsChannelGroupIdxType {
	return CommandProcessorsChannelGroupIdxType(getChannelGroupIdx(hvmCfg, ProcessorChannel_Command))
}

func provideProcessorChannelGroupIdxQuery(hvmCfg *HVMConfig) QueryProcessorsChannelGroupIdxType {
	return QueryProcessorsChannelGroupIdxType(getChannelGroupIdx(hvmCfg, ProcessorChannel_Query))
}

func getChannelGroupIdx(hvmCfg *HVMConfig, channelType ProcessorChannelType) int {
	for channelGroup, pc := range hvmCfg.processorsChannels {
		if pc.ChannelType == channelType {
			return channelGroup
		}
	}
	panic("wrong processor channel group config")
}

func provideChannelGroups(cfg *HVMConfig) (res []iprocbusmem.ChannelGroup) {
	for _, pc := range cfg.processorsChannels {
		res = append(res, pc.ChannelGroup)
	}
	return
}

func provideCachingAppStorageProvider(hvmCfg *HVMConfig, storageCacheSize StorageCacheSizeType, metrics imetrics.IMetrics,
	hvmName commandprocessor.HVMName, uncachingProivder IAppStorageUncachingProviderFactory) (istorage.IAppStorageProvider, error) {
	aspNonCaching := uncachingProivder()
	res := istoragecache.Provide(int(storageCacheSize), aspNonCaching, metrics, string(hvmName))
	return res, nil
}

// синхронный актуализатор один на приложение из-за storages, которые у каждого приложения свои
// сделаем так, чтобы в командный процессор подавался свитч по appName, который выберет нужный актуализатор с нужным набором проекторов
type switchByAppName struct {
}

func (s *switchByAppName) Switch(work interface{}) (branchName string, err error) {
	return work.(interface{ AppQName() istructs.AppQName }).AppQName().String(), nil
}

func provideSyncActualizerFactory(hvmApps HVMApps, structsProvider istructs.IAppStructsProvider, n10nBroker in10n.IN10nBroker, mpq MaxPrepareQueriesType, actualizerFactory projectors.SyncActualizerFactory, secretReader isecrets.ISecretReader) commandprocessor.SyncActualizerFactory {
	return func(hvmCtx context.Context, partitionID istructs.PartitionID) pipeline.ISyncOperator {
		actualizers := []pipeline.SwitchOperatorOptionFunc{}
		for _, appQName := range hvmApps {
			appStructs, err := structsProvider.AppStructs(appQName)
			if err != nil {
				panic(err)
			}
			if len(appStructs.SyncProjectors()) == 0 {
				actualizers = append(actualizers, pipeline.SwitchBranch(appQName.String(), &pipeline.NOOP{}))
				continue
			}
			conf := projectors.SyncActualizerConf{
				Ctx: hvmCtx,
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

func provideBlobStorage(bas BlobAppStorage, nowFunc func() time.Time) BlobStorage {
	return iblobstoragestg.Provide(bas, nowFunc)
}

func provideRouterAppStorage(astp istorage.IAppStorageProvider) (dbcertcache.RouterAppStorage, error) {
	return astp.AppStorage(istructs.AppQName_sys_router)
}

// port 80 -> [0] is http server, port 443 -> [0] is https server, [1] is acme server
func provideRouterServices(hvmCtx context.Context, rp router.RouterParams, busTimeout BusTimeout, broker in10n.IN10nBroker, quotas in10n.Quotas,
	nowFunc func() time.Time, bsc router.BlobberServiceChannels, bms router.BLOBMaxSizeType, blobberClusterAppID BlobberAppClusterID, blobStorage BlobStorage,
	routerAppStorage dbcertcache.RouterAppStorage, autocertCache autocert.Cache, bus ibus.IBus, hvmPortSource *HVMPortSource, appsWSAmounts map[istructs.AppQName]istructs.AppWSAmount) RouterServices {
	bp := &router.BlobberParams{
		ClusterAppBlobberID:    uint32(blobberClusterAppID),
		ServiceChannels:        bsc,
		BLOBStorage:            blobStorage,
		BLOBWorkersNum:         DefaultBLOBWorkersNum,
		RetryAfterSecondsOn503: DefaultRetryAfterSecondsOn503,
		BLOBMaxSize:            bms,
	}
	res := router.ProvideBP3(hvmCtx, rp, time.Duration(busTimeout), broker, quotas, bp, autocertCache, bus, appsWSAmounts)
	hvmPortSource.getter = func() HVMPortType {
		return HVMPortType(res[0].(interface{ GetPort() int }).GetPort())
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
	imetrics imetrics.IMetrics, hvm commandprocessor.HVMName, mpq MaxPrepareQueriesType, authn iauthnz.IAuthenticator, authz iauthnz.IAuthorizer,
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
			string(hvm), authn, authz, appCfgs)))
	}
	return pipeline.ForkOperator(pipeline.ForkSame, forks[0], forks[1:]...)
}

func provideCommandProcessors(cpCount CommandProcessorsCount, ccf CommandChannelFactory, cpFactory commandprocessor.ServiceFactory) OperatorCommandProcessors {
	forks := make([]pipeline.ForkOperatorOptionFunc, cpCount)
	for i := 0; i < int(cpCount); i++ {
		forks[i] = pipeline.ForkBranch(pipeline.ServiceOperator(cpFactory(ccf(i), istructs.PartitionID(i))))
	}
	return pipeline.ForkOperator(pipeline.ForkSame, forks[0], forks[1:]...)
}

func provideAsyncActualizersFactory(appStructsProvider istructs.IAppStructsProvider, n10nBroker in10n.IN10nBroker, asyncActualizerFactory projectors.AsyncActualizerFactory, secretReader isecrets.ISecretReader) AsyncActualizersFactory {
	return func(hvmCtx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories, partitionID istructs.PartitionID, opts []state.ActualizerStateOptFunc) pipeline.ISyncOperator {
		var asyncProjectors []pipeline.ForkOperatorOptionFunc
		appStructs, err := appStructsProvider.AppStructs(appQName)
		if err != nil {
			panic(err)
		}

		conf := projectors.AsyncActualizerConf{
			Ctx:      hvmCtx,
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
	return func(hvmCtx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories, partitionID istructs.PartitionID) pipeline.ISyncOperator {
		return aaf(hvmCtx, appQName, asyncProjectorFactories, partitionID, opts)
	}
}

func provideAppServiceFactory(apf AppPartitionFactory, pa AppPartitionsCount) AppServiceFactory {
	return func(hvmCtx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories) pipeline.ISyncOperator {
		forks := make([]pipeline.ForkOperatorOptionFunc, pa)
		for i := 0; i < int(pa); i++ {
			forks[i] = pipeline.ForkBranch(apf(hvmCtx, appQName, asyncProjectorFactories, istructs.PartitionID(i)))
		}
		return pipeline.ForkOperator(pipeline.ForkSame, forks[0], forks[1:]...)
	}
}

func provideOperatorAppServices(apf AppServiceFactory, hvmApps HVMApps, asp istructs.IAppStructsProvider) OperatorAppServicesFactory {
	return func(hvmCtx context.Context) pipeline.ISyncOperator {
		var branches []pipeline.ForkOperatorOptionFunc
		for _, appQName := range hvmApps {
			as, err := asp.AppStructs(appQName)
			if err != nil {
				panic(err)
			}
			if len(as.AsyncProjectors()) == 0 {
				continue
			}
			branch := pipeline.ForkBranch(apf(hvmCtx, appQName, as.AsyncProjectors()))
			branches = append(branches, branch)
		}
		if len(branches) == 0 {
			return &pipeline.NOOP{}
		}
		return pipeline.ForkOperator(pipeline.ForkSame, branches[0], branches[1:]...)
	}
}

func provideServicePipeline(hvmCtx context.Context, opCommandProcessors OperatorCommandProcessors, opQueryProcessors OperatorQueryProcessors, opAppServices OperatorAppServicesFactory,
	routerServiceOp RouterServiceOperator, metricsServiceOp MetricsServiceOperator) ServicePipeline {
	return pipeline.NewSyncPipeline(hvmCtx, "ServicePipeline",
		pipeline.WireSyncOperator("service fork operator", pipeline.ForkOperator(pipeline.ForkSame,

			// HVM
			pipeline.ForkBranch(pipeline.ForkOperator(pipeline.ForkSame,
				pipeline.ForkBranch(opQueryProcessors),
				pipeline.ForkBranch(opCommandProcessors),
				pipeline.ForkBranch(opAppServices(hvmCtx)), // hvmCtx here is for AsyncActualizerConf at AsyncActualizerFactory only
			)),

			// Router
			// hvmCtx here is for blobber service to stop reading from ServiceChannel on HVM shutdown
			pipeline.ForkBranch(routerServiceOp),

			// Metrics http service
			pipeline.ForkBranch(metricsServiceOp),
		)),
	)
}
