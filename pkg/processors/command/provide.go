/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type workspace struct {
	NextWLogOffset istructs.Offset
	idGenerator    istructs.IIDGenerator
}

type cmdProc struct {
	pNumber       istructs.PartitionID
	appPartition  *appPartition
	appPartitions map[appdef.AppQName]*appPartition
	n10nBroker    in10n.IN10nBroker
	now           coreutils.TimeFunc
	authenticator iauthnz.IAuthenticator
	authorizer    iauthnz.IAuthorizer
	storeOp       pipeline.ISyncOperator
}

type appPartition struct {
	workspaces     map[istructs.WSID]*workspace
	nextPLogOffset istructs.Offset
}

// syncActualizerFactory - это фабрика(разделИД), которая возвращает свитч, в бранчах которого по синхронному актуализатору на каждое приложение, внутри каждого - проекторы на каждое приложение
func ProvideServiceFactory(appParts appparts.IAppPartitions, now coreutils.TimeFunc,
	n10nBroker in10n.IN10nBroker, metrics imetrics.IMetrics, vvm VVMName, authenticator iauthnz.IAuthenticator, authorizer iauthnz.IAuthorizer,
	secretReader isecrets.ISecretReader) ServiceFactory {
	return func(commandsChannel CommandChannel, partitionID istructs.PartitionID) pipeline.IService {
		cmdProc := &cmdProc{
			pNumber:       partitionID,
			appPartitions: map[appdef.AppQName]*appPartition{},
			n10nBroker:    n10nBroker,
			now:           now,
			authenticator: authenticator,
			authorizer:    authorizer,
		}

		return pipeline.NewService(func(vvmCtx context.Context) {
			hsp := newHostStateProvider(vvmCtx, partitionID, secretReader)
			cmdProc.storeOp = pipeline.NewSyncPipeline(vvmCtx, "store",
				pipeline.WireFunc("applyRecords", func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
					// sync apply records
					cmd := work.(*cmdWorkpiece)
					if err = cmd.appStructs.Records().Apply(cmd.pLogEvent); err != nil {
						cmd.appPartitionRestartScheduled = true
					}
					return err
				}), pipeline.WireSyncOperator("syncProjectorsAndPutWLog", pipeline.ForkOperator(pipeline.ForkSame,
					// forK: sync projector and PutWLog

					pipeline.ForkBranch(
						pipeline.NewSyncOp(func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
							cmd := work.(*cmdWorkpiece)
							cmd.syncProjectorsStart = time.Now()
							err = cmd.appPart.DoSyncActualizer(ctx, work)
							cmd.metrics.increase(ProjectorsSeconds, time.Since(cmd.syncProjectorsStart).Seconds())
							cmd.syncProjectorsStart = time.Time{}
							if err != nil {
								cmd.appPartitionRestartScheduled = true
							}

							return err
						}),
					),

					pipeline.ForkBranch(pipeline.NewSyncOp(func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
						// put WLog
						cmd := work.(*cmdWorkpiece)
						if err = cmd.appStructs.Events().PutWlog(cmd.pLogEvent); err != nil {
							cmd.appPartitionRestartScheduled = true
						} else {
							cmd.workspace.NextWLogOffset++
						}
						return
					})),
				)))
			cmdPipeline := pipeline.NewSyncPipeline(vvmCtx, "Command Processor",
				pipeline.WireFunc("borrowAppPart", borrowAppPart),
				pipeline.WireFunc("limitCallRate", limitCallRate),
				pipeline.WireFunc("getWSDesc", getWSDesc),
				pipeline.WireFunc("authenticate", cmdProc.authenticate),
				pipeline.WireFunc("checkWSInitialized", checkWSInitialized),
				pipeline.WireFunc("checkWSActive", checkWSActive),
				pipeline.WireFunc("getIWorkspace", getIWorkspace),
				pipeline.WireFunc("getAppPartition", cmdProc.getAppPartition),
				pipeline.WireFunc("getICommand", getICommand),
				pipeline.WireFunc("authorizeRequest", cmdProc.authorizeRequest),
				pipeline.WireFunc("unmarshalRequestBody", unmarshalRequestBody),
				pipeline.WireFunc("getWorkspace", cmdProc.getWorkspace),
				pipeline.WireFunc("getRawEventBuilderBuilders", cmdProc.getRawEventBuilder),
				pipeline.WireFunc("getArgsObject", getArgsObject),
				pipeline.WireFunc("getUnloggedArgsObject", getUnloggedArgsObject),
				pipeline.WireFunc("checkArgsRefIntegrity", checkArgsRefIntegrity),
				pipeline.WireFunc("parseCUDs", parseCUDs),
				pipeline.WireFunc("checkCUDsAllowed", checkCUDsAllowed),
				pipeline.WireSyncOperator("wrongArgsCatcher", &wrongArgsCatcher{}), // any error before -> wrap error into bad request http error
				pipeline.WireFunc("authorizeCUDs", cmdProc.authorizeCUDs),
				pipeline.WireFunc("checkIsActiveinCUDs", checkIsActiveInCUDs),
				pipeline.WireFunc("writeCUDs", cmdProc.writeCUDs),
				pipeline.WireFunc("getCmdResultBuilder", cmdProc.getCmdResultBuilder),
				pipeline.WireFunc("buildCommandArgs", cmdProc.buildCommandArgs),
				pipeline.WireFunc("execCommand", execCommand),
				pipeline.WireFunc("checkResponseIntent", checkResponseIntent),
				pipeline.WireFunc("build raw event", buildRawEvent),
				pipeline.WireFunc("eventValidators", cmdProc.eventValidators),
				pipeline.WireFunc("validateCUDsQNames", cmdProc.validateCUDsQNames),
				pipeline.WireFunc("cudsValidators", cmdProc.cudsValidators),
				pipeline.WireFunc("validateCmdResult", validateCmdResult),
				pipeline.WireFunc("getIDGenerator", getIDGenerator),
				pipeline.WireFunc("putPLog", cmdProc.putPLog),
				pipeline.WireFunc("store", cmdProc.storeOp.DoSync),
				pipeline.WireFunc("n10n", cmdProc.n10n),
			)
			// TODO: сделать потом plogOffset свой по каждому разделу, wlogoffset - свой для каждого wsid
			defer cmdPipeline.Close()
			for vvmCtx.Err() == nil {
				select {
				case intf := <-commandsChannel:
					start := time.Now()
					cmdMes := intf.(ICommandMessage)
					cmd := &cmdWorkpiece{
						cmdMes:            cmdMes,
						requestData:       coreutils.MapObject{},
						appParts:          appParts,
						hostStateProvider: hsp,
						metrics: commandProcessorMetrics{
							vvmName: string(vvm),
							app:     cmdMes.AppQName(),
							metrics: metrics,
						},
					}
					func() { // borrowed application partition should be guaranteed to be freed
						defer cmd.Release()
						cmd.metrics.increase(CommandsTotal, 1.0)
						cmdHandlingErr := cmdPipeline.SendSync(cmd)
						if cmdHandlingErr != nil {
							logger.Error(cmdHandlingErr)
						}
						sendResponse(cmd, cmdHandlingErr)
						if cmd.appPartitionRestartScheduled {
							logger.Info(fmt.Sprintf("partition %d will be restarted due of an error on writing to Log: %s", cmd.cmdMes.PartitionID(), cmdHandlingErr))
							delete(cmdProc.appPartitions, cmd.cmdMes.AppQName())
						}
					}()
					metrics.IncreaseApp(CommandsSeconds, string(vvm), cmdMes.AppQName(), time.Since(start).Seconds())
				case <-vvmCtx.Done():
					cmdProc.appPartitions = map[appdef.AppQName]*appPartition{} // clear appPartitions to test recovery
					return
				}
			}
		})
	}
}
