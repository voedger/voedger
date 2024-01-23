/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"time"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type workspace struct {
	NextWLogOffset istructs.Offset
	idGenerator    istructs.IIDGenerator
	iws            appdef.IWorkspace
}

type cmdProc struct {
	pNumber       istructs.PartitionID
	appPartition  *appPartition
	appPartitions map[istructs.AppQName]*appPartition
	n10nBroker    in10n.IN10nBroker
	now           coreutils.TimeFunc
	authenticator iauthnz.IAuthenticator
	authorizer    iauthnz.IAuthorizer
	cfgs          istructsmem.AppConfigsType
}

type appPartition struct {
	workspaces     map[istructs.WSID]*workspace
	nextPLogOffset istructs.Offset
}

// syncActualizerFactory - это фабрика(разделИД), которая возвращает свитч, в бранчах которого по синхронному актуализатору на каждое приложение, внутри каждого - проекторы на каждое приложение
func ProvideServiceFactory(appParts appparts.IAppPartitions, now coreutils.TimeFunc, syncActualizerFactory SyncActualizerFactory,
	n10nBroker in10n.IN10nBroker, metrics imetrics.IMetrics, vvm VVMName, authenticator iauthnz.IAuthenticator, authorizer iauthnz.IAuthorizer,
	secretReader isecrets.ISecretReader, appConfigsType istructsmem.AppConfigsType) ServiceFactory {
	return func(commandsChannel CommandChannel, partitionID istructs.PartitionID) pipeline.IService {
		cmdProc := &cmdProc{
			pNumber:       partitionID,
			appPartitions: map[istructs.AppQName]*appPartition{},
			n10nBroker:    n10nBroker,
			now:           now,
			authenticator: authenticator,
			authorizer:    authorizer,
			cfgs:          appConfigsType,
		}

		return pipeline.NewService(func(vvmCtx context.Context) {
			hsp := newHostStateProvider(vvmCtx, partitionID, secretReader)
			syncActualizerFactory := syncActualizerFactory(vvmCtx, partitionID)
			cmdPipeline := pipeline.NewSyncPipeline(vvmCtx, "Command Processor",
				pipeline.WireFunc("borrowAppPart", borrowAppPart),
				pipeline.WireFunc("limitCallRate", limitCallRate),
				pipeline.WireFunc("getWSDesc", getWSDesc),
				pipeline.WireFunc("authenticate", cmdProc.authenticate),
				pipeline.WireFunc("checkWSInitialized", checkWSInitialized),
				pipeline.WireFunc("checkWSActive", checkWSActive),
				pipeline.WireFunc("getAppPartition", cmdProc.provideGetAppPartition(syncActualizerFactory)),
				pipeline.WireFunc("getIWorkspace", getIWorkspace),
				pipeline.WireFunc("getICommand", getICommand),
				pipeline.WireFunc("getResources", getResources),
				pipeline.WireFunc("getExec", getExec),
				pipeline.WireFunc("authorizeRequest", cmdProc.authorizeRequest),
				pipeline.WireFunc("unmarshalRequestBody", unmarshalRequestBody),
				pipeline.WireFunc("getWorkspace", cmdProc.getWorkspace),
				pipeline.WireFunc("getRawEventBuilderBuilders", cmdProc.getRawEventBuilder),
				pipeline.WireFunc("getArgsObject", getArgsObject),
				pipeline.WireFunc("getUnloggedArgsObject", getUnloggedArgsObject),
				pipeline.WireFunc("checkArgsRefIntegrity", checkArgsRefIntegrity),
				pipeline.WireFunc("parseCUDs", parseCUDs),
				pipeline.WireSyncOperator("wrongArgsCatcher", &wrongArgsCatcher{}), // any error before -> wrap error into bad request http error
				pipeline.WireFunc("authorizeCUDs", cmdProc.authorizeCUDs),
				pipeline.WireFunc("checkIsActiveinCUDs", checkIsActiveInCUDs),
				pipeline.WireFunc("writeCUDs", cmdProc.writeCUDs),
				pipeline.WireFunc("getCmdResultBuilder", cmdProc.getCmdResultBuilder),
				pipeline.WireFunc("buildCommandArgs", cmdProc.buildCommandArgs),
				pipeline.WireFunc("execCommand", execCommand),
				pipeline.WireFunc("build raw event", buildRawEvent),
				pipeline.WireFunc("validate", cmdProc.validate),
				pipeline.WireFunc("validateCmdResult", validateCmdResult),
				pipeline.WireFunc("getIDGenerator", getIDGenerator),
				pipeline.WireFunc("putPLog", cmdProc.putPLog),
				pipeline.WireFunc("applyRecords", applyRecords),
				pipeline.WireFunc("syncProjectorsStart", syncProjectorsBegin),
				pipeline.WireFunc("syncProjectors", provideSyncActualizerFactory(syncActualizerFactory)),
				pipeline.WireFunc("syncProjectorsEnd", syncProjectorsEnd),
				pipeline.WireFunc("n10n", cmdProc.n10n),
				pipeline.WireFunc("putWLog", putWLog),
				pipeline.WireSyncOperator("sendResponse", &opSendResponse{cmdProc: cmdProc}),  // ICatch
				pipeline.FinallyOperator[*cmdWorkpiece]("releaseWorkpiece", releaseWorkpiece), // ICatch
			)
			// TODO: сделать потом plogOffset свой по каждому разделу, wlogoffset - свой для каждого wsid
			defer cmdPipeline.Close()
			for vvmCtx.Err() == nil {
				select {
				case intf := <-commandsChannel:
					start := time.Now()
					cmd := &cmdWorkpiece{
						cmdMes:            intf.(ICommandMessage),
						requestData:       coreutils.MapObject{},
						appParts:          appParts,
						hostStateProvider: hsp,
					}
					cmd.metrics = commandProcessorMetrics{
						vvm:     string(vvm),
						app:     cmd.cmdMes.AppQName(),
						metrics: metrics,
					}
					cmd.metrics.increase(CommandsTotal, 1.0)
					if err := cmdPipeline.SendSync(cmd); err != nil {
						logger.Error("unhandled error: " + err.Error())
					}
					cmd.metrics.increase(CommandsSeconds, time.Since(start).Seconds())
				case <-vvmCtx.Done():
					cmdProc.appPartitions = map[istructs.AppQName]*appPartition{} // clear appPartitions to test recovery
					return
				}
			}
		})
	}
}
