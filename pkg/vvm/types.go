/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/router"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/workspace"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	"github.com/voedger/voedger/pkg/vvm/metrics"
)

type ServicePipeline pipeline.ISyncPipeline
type OperatorCommandProcessors pipeline.ISyncOperator
type OperatorCommandProcessor pipeline.ISyncOperator
type OperatorQueryProcessors pipeline.ISyncOperator
type OperatorBLOBProcessors pipeline.ISyncOperator
type OperatorQueryProcessor pipeline.ISyncOperator
type AppPartitionFactory func(ctx context.Context, appQName appdef.AppQName, asyncProjectors istructs.Projectors, partitionID istructs.PartitionID) pipeline.ISyncOperator
type AsyncActualizersFactory func(ctx context.Context, appQName appdef.AppQName, asyncProjectors istructs.Projectors, partitionID istructs.PartitionID,
	tokens itokens.ITokens, federation federation.IFederation, opts []state.StateOptFunc) pipeline.ISyncOperator
type OperatorAppServicesFactory func(ctx context.Context) pipeline.ISyncOperator
type CommandChannelFactory func(channelIdx uint) commandprocessor.CommandChannel
type QueryChannel iprocbus.ServiceChannel
type AdminEndpointServiceOperator pipeline.ISyncOperator
type PublicEndpointServiceOperator pipeline.ISyncOperator
type BlobberAppClusterID istructs.ClusterAppID
type BlobStorage iblobstorage.IBLOBStorage
type BlobberAppStruct istructs.IAppStructs
type CommandProcessorsChannelGroupIdxType uint
type QueryProcessorsChannelGroupIdxType uint
type BLOBProcessorsChannelGroupIdxType uint
type MaxPrepareQueriesType int
type ServiceChannelFactory func(pcgt ProcessorChannelType, channelIdx uint) iprocbus.ServiceChannel
type AppStorageFactory func(appQName appdef.AppQName, appStorage istorage.IAppStorage) istorage.IAppStorage
type StorageCacheSizeType int
type VVMApps []appdef.AppQName
type BuiltInAppPackages struct {
	appparts.BuiltInApp
	Packages []parser.PackageFS // need for build baseline schemas
}
type AppConfigsTypeEmpty istructsmem.AppConfigsType
type BootstrapOperator pipeline.ISyncOperator
type SidecarAppsDefs map[appdef.AppQName]BuiltInAppPackages

type BuiltInAppsArtefacts struct {
	istructsmem.AppConfigsType
	builtInAppPackages []BuiltInAppPackages
}

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
	router.IAdminService
}
type MetricsServiceOperator pipeline.ISyncOperator
type MetricsServicePortInitial int
type VVMPortSource struct {
	getter      func() VVMPortType
	adminGetter func() int
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

type VVMAppsBuilder map[appdef.AppQName]builtinapps.Builder

type VVM struct {
	ServicePipeline
	builtinapps.APIs
	appparts.IAppPartitions
	AppsExtensionPoints map[appdef.AppQName]extensionpoints.IExtensionPoint
	MetricsServicePort  func() metrics.MetricsServicePort
	BuiltInAppsPackages []BuiltInAppPackages
}

type AppsExtensionPoints map[appdef.AppQName]extensionpoints.IExtensionPoint

type VVMConfig struct {
	VVMAppsBuilder             VVMAppsBuilder // is a map
	Time                       coreutils.ITime
	RouterWriteTimeout         int
	RouterReadTimeout          int
	RouterConnectionsLimit     int
	RouterHTTP01ChallengeHosts []string
	RouteDefault               string
	Routes                     map[string]string
	RoutesRewrite              map[string]string
	RouteDomains               map[string]string
	SendTimeout                bus.SendTimeout
	StorageFactory             func() (provider istorage.IAppStorageFactory, err error)
	BLOBMaxSize                iblobstorage.BLOBMaxSizeType
	Name                       processors.VVMName
	NumCommandProcessors       istructs.NumCommandProcessors
	NumQueryProcessors         istructs.NumQueryProcessors
	NumBLOBProcessors          istructs.NumBLOBProcessors
	MaxPrepareQueries          MaxPrepareQueriesType
	StorageCacheSize           StorageCacheSizeType
	processorsChannels         []ProcesorChannel
	ActualizerStateOpts        []state.StateOptFunc
	SecretsReader              isecrets.ISecretReader
	SmtpConfig                 smtp.Cfg
	WSPostInitFunc             workspace.WSPostInitFunc
	DataPath                   string
	MetricsServicePort         MetricsServicePortInitial

	// 0 -> dynamic port will be used, new on each vvmIdx
	// >0 -> vVMPort+vvmIdx will be actually used
	VVMPort VVMPortType

	// test and FederationURL contains port -> the port will be relaced with the actual VVMPort
	FederationURL *url.URL

	// used in tests only
	// normally is empty in VIT. coretuils.IsTest -> UUID is added to the keyspace name at istorage/provider/Provide()
	// need to e.g. test VVM restart preserving storage
	KeyspaceNameSuffix string
}

type VoedgerVM struct {
	*VVM
	vvmCtxCancel func()
	vvmCleanup   func()
}

type ignition struct{}

func (i ignition) Release() {}
