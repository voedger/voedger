/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package vvm

import (
	"context"
	"net"
	"net/url"
	"runtime/debug"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/ielections"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/isequencer"
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
	"github.com/voedger/voedger/pkg/vvm/storage"
)

type ServicePipeline pipeline.ISyncPipeline
type OperatorCommandProcessors pipeline.ISyncOperator
type OperatorCommandProcessor pipeline.ISyncOperator
type OperatorQueryProcessors_V1 pipeline.ISyncOperator
type OperatorQueryProcessors_V2 pipeline.ISyncOperator
type OperatorBLOBProcessors pipeline.ISyncOperator
type OperatorQueryProcessor pipeline.ISyncOperator
type AppPartitionFactory func(ctx context.Context, appQName appdef.AppQName, asyncProjectors istructs.Projectors, partitionID istructs.PartitionID) pipeline.ISyncOperator
type AsyncActualizersFactory func(ctx context.Context, appQName appdef.AppQName, asyncProjectors istructs.Projectors, partitionID istructs.PartitionID,
	tokens itokens.ITokens, federation federation.IFederation, stateOpts state.StateOpts) pipeline.ISyncOperator
type OperatorAppServicesFactory func(ctx context.Context) pipeline.ISyncOperator
type CommandChannelFactory func(channelIdx uint) commandprocessor.CommandChannel
type QueryChannel_V1 iprocbus.ServiceChannel
type QueryChannel_V2 iprocbus.ServiceChannel
type AdminEndpointServiceOperator pipeline.ISyncOperator
type PublicEndpointServiceOperator pipeline.ISyncOperator
type BlobberAppClusterID istructs.ClusterAppID
type BlobberAppStruct istructs.IAppStructs
type CommandProcessorsChannelGroupIdxType uint
type QueryProcessorsChannelGroupIdxType_V1 uint
type QueryProcessorsChannelGroupIdxType_V2 uint
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
type VVMPortSource struct {
	getter      func() VVMPortType
	adminGetter func() int
}
type IAppStorageUncachingProviderFactory func() (provider istorage.IAppStorageProvider)
type AppPartsCtlPipelineService struct {
	apppartsctl.IAppPartitionsController
}
type IAppPartsCtlPipelineService pipeline.IService

type NumVVM = uint32

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
type LeadershipAcquisitionDuration time.Duration

type VVM struct {
	ServicePipeline
	builtinapps.APIs
	appparts.IAppPartitions
	AppsExtensionPoints map[appdef.AppQName]extensionpoints.IExtensionPoint
	MetricsServicePort  func() metrics.MetricsServicePort
	BuiltInAppsPackages []BuiltInAppPackages
	TTLStorage          ielections.ITTLStorage[storage.TTLStorageImplKey, string]
	BuildInfo           *debug.BuildInfo
}

type AppsExtensionPoints map[appdef.AppQName]extensionpoints.IExtensionPoint

type VVMConfig struct {
	VVMAppsBuilder                   VVMAppsBuilder // is a map
	Time                             timeu.ITime
	RouterWriteTimeout               int
	RouterReadTimeout                int
	RouterConnectionsLimit           int
	RouterHTTP01ChallengeHosts       []string
	RouteDefault                     string
	Routes                           map[string]string
	RoutesRewrite                    map[string]string
	RouteDomains                     map[string]string
	SendTimeout                      bus.SendTimeout
	StorageFactory                   func(time timeu.ITime) (provider istorage.IAppStorageFactory, err error)
	BLOBMaxSize                      iblobstorage.BLOBMaxSizeType
	Name                             processors.VVMName
	NumCommandProcessors             istructs.NumCommandProcessors
	NumQueryProcessors               istructs.NumQueryProcessors
	NumBLOBProcessors                istructs.NumBLOBProcessors
	MaxPrepareQueries                MaxPrepareQueriesType
	StorageCacheSize                 StorageCacheSizeType
	processorsChannels               []ProcesorChannel
	EmailSender                      state.IEmailSender
	SecretsReader                    isecrets.ISecretReader
	SMTPConfig                       smtp.Cfg
	WSPostInitFunc                   workspace.WSPostInitFunc
	DataPath                         string
	MetricsServicePort               metrics.MetricsServicePort
	AdminPort                        int
	SchemasCache                     ISchemasCache // normally NullSchemasCache in production, vit.SysAppsSchemasCache in VIT tests
	PolicyOptsForFederationWithRetry federation.PolicyOptsForWithRetry

	// 0 -> dynamic port will be used, new on each vvmIdx
	// >0 -> vVMPort+vvmIdx will be actually used
	VVMPort VVMPortType

	// test and FederationURL contains port -> the port will be relaced with the actual VVMPort
	FederationURL *url.URL

	// the string that will be added to the keyspace names of each app to isolate keyspaces among few integration tests run
	// simultaneously on the same storage driver.
	// normally should be a random string matching ^[a-z0-9]+$
	// use [provider.NewTestKeyspaceIsolationSuffix()]
	// normally should be used in tests only
	// exposed here for test cases when >1 VVMs must use the same storage
	KeyspaceIsolationSuffix string

	// [~server.design.orch/VVMConfig.Orch~impl]
	NumVVM NumVVM // amount of VVMs in the cluster. Default 1
	IP     net.IP // current IP of the VVM. Used as the value for leaderhsip elections

	// [~server.design.sequences/cmp.VVMConfig.SequencesTrustLevel~impl]
	SequencesTrustLevel isequencer.SequencesTrustLevel
}

type VoedgerVM struct {
	*VVM
	vvmCtxCancel     func()
	vvmCleanup       func()
	electionsCleanup func()

	// closed when some problem occurs, VVM terminates itself due to leadership loss or problems with the launching
	problemCtx       context.Context
	problemCtxCancel context.CancelFunc
	problemErrCh     chan error

	// used to ensure we publish the error only once
	problemCtxErrOnce sync.Once

	// closed when VVM should be stopped outside
	vvmShutCtx       context.Context
	vvmShutCtxCancel context.CancelFunc

	// closed when VVM services should be stopped (but LeadershipMonitor)
	servicesShutCtx       context.Context
	servicesShutCtxCancel context.CancelFunc

	// closed after all services are stopped and LeadershipMonitor should be stopped
	monitorShutCtx       context.Context
	monitorShutCtxCancel context.CancelFunc
	monitorShutWg        sync.WaitGroup

	// closed after all (services and LeadershipMonitor) is stopped
	shutdownedCtx       context.Context
	shutdownedCtxCancel context.CancelFunc
	numVVM              NumVVM
	ip                  net.IP
	leadershipCtx       context.Context

	// used in tests only
	leadershipAcquisitionTimerArmed chan struct{}
}

type IVVMElections ielections.IElections[storage.TTLStorageImplKey, string]

type ignition struct{}

func (i ignition) Release() {}

type ISchemasCache interface {
	// @ConcurrentAccess
	// must return nil if not found
	Get(appQName appdef.AppQName) *parser.AppSchemaAST
	// @ConcurrentAccess
	Put(appQName appdef.AppQName, schema *parser.AppSchemaAST)
}

type NullSchemasCache struct{}

func (*NullSchemasCache) Get(appQName appdef.AppQName) *parser.AppSchemaAST {
	return nil
}

func (*NullSchemasCache) Put(appQName appdef.AppQName, schema *parser.AppSchemaAST) {}
