/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"net/url"
	"time"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/pipeline"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/router"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vvm/metrics"
)

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
type MaxPrepareQueriesType int
type ServiceChannelFactory func(pcgt ProcessorChannelType, channelIdx int) iprocbus.ServiceChannel
type AppStorageFactory func(appQName istructs.AppQName, appStorage istorage.IAppStorage) istorage.IAppStorage
type StorageCacheSizeType int
type VVMApps []istructs.AppQName
type BuiltInAppsPackages struct {
	apppartsctl.BuiltInApp
	Packages []parser.PackageFS
}

type BusTimeout time.Duration
type FederationURL func() *url.URL
type VVMIdxType int
type VVMPortType int
type ProcessorChannelType int
type ProcesorChannel struct {
	iprocbusmem.ChannelGroup
	ChannelType ProcessorChannelType
}

type RouterServices struct {
	router.IHTTPService
	router.IACMEService
}
type MetricsServiceOperator pipeline.ISyncOperator
type MetricsServicePortInitial int
type VVMPortSource struct {
	getter func() VVMPortType
}
type IAppStorageUncachingProviderFactory func() (provider istorage.IAppStorageProvider)
type AppPartsCtlPipelineService struct {
	apppartsctl.IAppPartitionsController
}
type IAppPartsCtlPipelineService pipeline.IService

type PostDocFieldType struct {
	Kind              appdef.DataKind
	Required          bool
	VerificationKinds []appdef.VerificationKind // empty -> not verified
}

type PostDocDesc struct {
	Kind        appdef.TypeKind
	IsSingleton bool
}

type VVMAppsBuilder map[istructs.AppQName]apps.AppBuilder

type VVM struct {
	ServicePipeline
	apps.APIs
	AppsExtensionPoints map[istructs.AppQName]extensionpoints.IExtensionPoint
	MetricsServicePort  func() metrics.MetricsServicePort
	BuiltInAppsPackages []BuiltInAppsPackages
}

type AppsExtensionPoints map[istructs.AppQName]extensionpoints.IExtensionPoint

type VVMConfig struct {
	VVMAppsBuilder             VVMAppsBuilder // is a map
	TimeFunc                   coreutils.TimeFunc
	RouterWriteTimeout         int
	RouterReadTimeout          int
	RouterConnectionsLimit     int
	RouterHTTP01ChallengeHosts []string
	RouteDefault               string
	Routes                     map[string]string
	RoutesRewrite              map[string]string
	RouteDomains               map[string]string
	BusTimeout                 BusTimeout
	Quotas                     in10n.Quotas
	StorageFactory             func() (provider istorage.IAppStorageFactory, err error)
	BlobberServiceChannels     router.BlobberServiceChannels
	BLOBMaxSize                router.BLOBMaxSizeType
	Name                       commandprocessor.VVMName
	NumCommandProcessors       coreutils.CommandProcessorsCount
	NumQueryProcessors         coreutils.QueryProcessorsCount
	MaxPrepareQueries          MaxPrepareQueriesType
	StorageCacheSize           StorageCacheSizeType
	processorsChannels         []ProcesorChannel
	// 0 -> dynamic port will be used, new on each vvmIdx
	// >0 -> vVMPort+vvmIdx will be actually used
	VVMPort            VVMPortType
	MetricsServicePort MetricsServicePortInitial
	// test and FederationURL contains port -> the port will be relaced with the actual VVMPort
	FederationURL       *url.URL
	ActualizerStateOpts []state.ActualizerStateOptFunc
	SecretsReader       isecrets.ISecretReader
}

type resultSenderErrorFirst struct {
	ctx    context.Context
	sender ibus.ISender
	rs     ibus.IResultSenderClosable
}

type VoedgerVM struct {
	*VVM
	vvmCtxCancel func()
	vvmCleanup   func()
}
