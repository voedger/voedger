/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"net/url"
	"time"

	ibus "github.com/untillpro/airs-ibus"
	router "github.com/untillpro/airs-router2"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/pipeline"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/vvm/metrics"
)

type AppPartitionsCount int
type ServicePipeline pipeline.ISyncPipeline
type OperatorCommandProcessors pipeline.ISyncOperator
type OperatorCommandProcessor pipeline.ISyncOperator
type OperatorQueryProcessors pipeline.ISyncOperator
type OperatorQueryProcessor pipeline.ISyncOperator
type AppServiceFactory func(ctx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories) pipeline.ISyncOperator
type AppPartitionFactory func(ctx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories, partitionID istructs.PartitionID) pipeline.ISyncOperator
type AsyncActualizersFactory func(ctx context.Context, appQName istructs.AppQName, asyncProjectorFactories AsyncProjectorFactories, partitionID istructs.PartitionID, opts []state.ActualizerStateOptFunc) pipeline.ISyncOperator
type OperatorAppServicesFactory func(ctx context.Context) pipeline.ISyncOperator
type CommandChannelFactory func(channelIdx int) commandprocessor.CommandChannel
type QueryChannel iprocbus.ServiceChannel
type RouterServiceOperator pipeline.ISyncOperator
type AsyncProjectorFactories []istructs.ProjectorFactory
type SyncProjectorFactories []istructs.ProjectorFactory
type BlobberAppClusterID istructs.ClusterAppID
type BlobStorage iblobstorage.IBLOBStorage
type BlobAppStorage istorage.IAppStorage
type BlobberAppStruct istructs.IAppStructs
type CommandProcessorsChannelGroupIdxType int
type QueryProcessorsChannelGroupIdxType int
type CommandProcessorsAmountType int
type MaxPrepareQueriesType int
type ServiceChannelFactory func(pcgt ProcessorChannelType, channelIdx int) iprocbus.ServiceChannel
type AppStorageFactory func(appQName istructs.AppQName, appStorage istorage.IAppStorage) istorage.IAppStorage
type StorageCacheSizeType int
type HVMApps []istructs.AppQName
type QueryProcessorsCount int
type CommandProcessorsCount int
type BusTimeout time.Duration
type FederationURLType func() *url.URL
type HVMIdxType int
type HVMPortType int
type ProcessorChannelType int
type ProcesorChannel struct {
	iprocbusmem.ChannelGroup
	ChannelType ProcessorChannelType
}
type RouterServices []interface{}
type MetricsServiceOperator pipeline.ISyncOperator
type MetricsServicePortInitial int
type HVMPortSource struct {
	getter func() HVMPortType
}
type IAppStorageUncachingProviderFactory func() (provider istorage.IAppStorageProvider)
type EPKey string
type EKey interface{}

type PostDocFieldType struct {
	Kind              appdef.DataKind
	Required          bool
	VerificationKinds []appdef.VerificationKind // empty -> not verified
}

type PostDocDesc struct {
	Kind        appdef.DefKind
	IsSingleton bool
}

type HVMAppBuilder func(hvmCfg *HVMConfig, hvmAPI HVMAPI, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, sep IStandardExtensionPoints)
type HVMAppsBuilder map[istructs.AppQName][]HVMAppBuilder

type HVM struct {
	ServicePipeline
	HVMAPI
	HVMApps
	AppsExtensionPoints map[istructs.AppQName]IStandardExtensionPoints
	MetricsServicePort  func() metrics.MetricsServicePort
}

type HVMAPI struct {
	itokens.ITokens
	istructs.IAppStructsProvider
	istructsmem.AppConfigsType
	istorage.IAppStorageProvider
	payloads.IAppTokensFactory
	FederationURL FederationURLType
}

type AppsExtensionPoints map[istructs.AppQName]IStandardExtensionPoints

type IStandardExtensionPoints interface {
	EPWSTemplates() IEPWSTemplates
	EPJournalIndices() IEPJournalIndices
	EPJournalPredicates() IEPJournalPredicates
	ExtensionPoint(epKey EPKey) IExtensionPoint
}

type NamedExtension struct {
	key   EKey
	value interface{}
}

// val could be map[interface{}]interface{} or IExtensionPoint
type implIExtensionPoint struct {
	key   EKey
	exts  []interface{} // element could be any or NamedExtension or IExtensionPoint
	value interface{}
}

type IEPWSTemplates IExtensionPoint
type IEPJournalIndices IExtensionPoint
type IEPJournalPredicates IExtensionPoint

type IExtensionPoint interface {
	// optional value is never set or set once. Otherwise -> panic
	ExtensionPoint(eKey EKey, value ...interface{}) IExtensionPoint
	AddNamed(eKey EKey, value interface{})
	Add(value interface{})
	Find(eKey EKey) (val interface{}, ok bool)
	Iterate(callback func(eKey EKey, value interface{}))
	Value() interface{}
}

type standardExtensionPointsImpl struct {
	rootExtensionPoint *implIExtensionPoint
}

type HVMConfig struct {
	HVMAppsBuilder             HVMAppsBuilder // is a map
	TimeFunc                   func() time.Time
	RouterWriteTimeout         int
	RouterReadTimeout          int
	RouterConnectionsLimit     int
	RouterHTTP01ChallengeHosts []string
	RouteDefault               string
	Routes                     map[string]string
	RoutesRewrite              map[string]string
	RouteDomains               map[string]string
	PartitionsCount            AppPartitionsCount
	BusTimeout                 BusTimeout
	Quotas                     in10n.Quotas
	StorageFactory             func() (provider istorage.IAppStorageFactory, err error)
	BlobberServiceChannels     router.BlobberServiceChannels
	BLOBMaxSize                router.BLOBMaxSizeType
	Name                       commandprocessor.HVMName
	NumCommandProcessors       CommandProcessorsCount
	NumQueryProcessors         QueryProcessorsCount
	MaxPrepareQueries          MaxPrepareQueriesType
	StorageCacheSize           StorageCacheSizeType
	processorsChannels         []ProcesorChannel
	// 0 -> dynamic port will be used, new on each hvmIdx
	// >0 -> HVMPort+hvmIdx will be actually used
	HVMPort            HVMPortType
	MetricsServicePort MetricsServicePortInitial
	// test and FederationURL contains port -> the port will be relaced with the actual HVMPort
	FederationURL       *url.URL
	ActualizerStateOpts []state.ActualizerStateOptFunc
}

type resultSenderErrorFirst struct {
	ctx    context.Context
	sender interface{}
	rs     ibus.IResultSenderClosable
	bus    ibus.IBus
}

type HeeusVM struct {
	*HVM
	hvmCtxCancel func()
	hvmCleanup   func()
}

type testISecretReader struct {
	realSecretReader isecrets.ISecretReader
}
