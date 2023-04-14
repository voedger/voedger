/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"time"

	ibus "github.com/untillpro/airs-ibus"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/voedger/pkg/iauthnz"
	"github.com/untillpro/voedger/pkg/in10n"
	"github.com/untillpro/voedger/pkg/isecrets"
	"github.com/untillpro/voedger/pkg/istructs"
	istructsmem "github.com/untillpro/voedger/pkg/istructsmem"
	imetrics "github.com/untillpro/voedger/pkg/metrics"
	"github.com/untillpro/voedger/pkg/pipeline"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type workspace struct {
	NextWLogOffset        istructs.Offset
	NextBaseID            istructs.RecordID
	NextCDocCRecordBaseID istructs.RecordID
}

type cmdProc struct {
	pNumber       istructs.PartitionID
	appPartition  *appPartition
	appPartitions map[istructs.AppQName]*appPartition
	n10nBroker    in10n.IN10nBroker
	now           func() time.Time
	authenticator iauthnz.IAuthenticator
	authorizer    iauthnz.IAuthorizer
}

type appPartition struct {
	workspaces     map[istructs.WSID]*workspace
	nextPLogOffset istructs.Offset
}

func ProvideJSONFuncParamsSchema(cfg *istructsmem.AppConfigType) {
	cfg.Schemas.Add(istructs.QNameJSON, istructs.SchemaKind_Object).
		AddField(Field_JSONSchemaBody, istructs.DataKind_string, true)
}

// syncActualizerFactory - это фабрика(разделИД), которая возвращает свитч, в бранчах которого по синхронному актуализатору на каждое приложение, внутри каждого - проекторы на каждое приложение
func ProvideServiceFactory(bus ibus.IBus, asp istructs.IAppStructsProvider, now func() time.Time, syncActualizerFactory SyncActualizerFactory,
	n10nBroker in10n.IN10nBroker, metrics imetrics.IMetrics, hvm HVMName, authenticator iauthnz.IAuthenticator, authorizer iauthnz.IAuthorizer,
	secretReader isecrets.ISecretReader) ServiceFactory {
	return func(commandsChannel CommandChannel, partitionID istructs.PartitionID) pipeline.IService {
		cmdProc := &cmdProc{
			pNumber:       partitionID,
			appPartitions: map[istructs.AppQName]*appPartition{},
			n10nBroker:    n10nBroker,
			now:           now,
			authenticator: authenticator,
			authorizer:    authorizer,
		}
		return pipeline.NewService(func(hvmCtx context.Context) {
			hsp := newHostStateProvider(hvmCtx, partitionID, secretReader)
			cmdPipeline := pipeline.NewSyncPipeline(hvmCtx, "Command Processor",
				pipeline.WireFunc("getAppStructs", getAppStructs),
				pipeline.WireFunc("limitCallRate", limitCallRate),
				pipeline.WireFunc("getWSDesc", getWSDesc),
				pipeline.WireFunc("checkWSInitialized", checkWSInitialized),
				pipeline.WireFunc("getAppPartition", cmdProc.getAppPartition),
				pipeline.WireFunc("getFunction", getFunction),
				pipeline.WireFunc("authenticate", cmdProc.authenticate),
				pipeline.WireFunc("authorizeRequest", cmdProc.authorizeRequest),
				pipeline.WireFunc("unmarshalRequestBody", unmarshalRequestBody),
				pipeline.WireFunc("getWorkspace", cmdProc.getWorkspace),
				pipeline.WireFunc("getRawEventBuilderBuilders", cmdProc.getRawEventBuilder),
				pipeline.WireFunc("getArgsObject", getArgsObject),
				pipeline.WireFunc("getUnloggedArgsObject", getUnloggedArgsObject),
				pipeline.WireFunc("checkArgsRefIntegrity", checkArgsRefIntegrity),
				pipeline.WireFunc("parseCUDs", parseCUDs),
				pipeline.WireSyncOperator("wrongArgsCatcher", &wrongArgsCatcher{}), // any error before -> wrap error into bad request http error
				pipeline.WireFunc("checkWSDescUpdating", checkWorkspaceDescriptorUpdating),
				pipeline.WireFunc("authorizeCUDs", cmdProc.authorizeCUDs),
				pipeline.WireFunc("writeCUDs", cmdProc.writeCUDs),
				pipeline.WireFunc("buildCommandArgs", cmdProc.buildCommandArgs),
				pipeline.WireFunc("execCommand", execCommand),
				pipeline.WireFunc("build raw event", buildRawEvent),
				pipeline.WireFunc("validate", cmdProc.validate),
				pipeline.WireFunc("putPLog", cmdProc.putPLog),
				pipeline.WireFunc("applyPLogEvent", applyPLogEvent),
				pipeline.WireFunc("syncProjectorsStart", syncProjectorsBegin),
				pipeline.WireSyncOperator("syncProjectors", syncActualizerFactory(hvmCtx, partitionID)),
				pipeline.WireFunc("syncProjectorsEnd", syncProjectorsEnd),
				pipeline.WireFunc("n10n", cmdProc.n10n),
				pipeline.WireFunc("putWLog", putWLog),
				pipeline.WireSyncOperator("sendResponse", &opSendResponse{bus: bus}), // ICatch
			)
			// TODO: сделать потом plogOffset свой по каждому разделу, wlogoffset - свой для каждого wsid
			defer cmdPipeline.Close()
			for hvmCtx.Err() == nil {
				select {
				case intf := <-commandsChannel:
					start := time.Now()
					cmd := &cmdWorkpiece{
						cmdMes:            intf.(ICommandMessage),
						requestData:       coreutils.MapObject{},
						asp:               asp,
						generatedIDs:      map[istructs.RecordID]istructs.RecordID{},
						hostStateProvider: hsp,
					}
					cmd.metrics = commandProcessorMetrics{
						hvm:     string(hvm),
						app:     cmd.cmdMes.AppQName(),
						metrics: metrics,
					}
					cmd.metrics.increase(CommandsTotal, 1.0)
					if err := cmdPipeline.SendSync(cmd); err != nil {
						logger.Error("unhandled error: " + err.Error())
					}
					if cmd.pLogEvent != nil {
						cmd.pLogEvent.Release()
					}
					if cmd.wLogEvent != nil {
						cmd.wLogEvent.Release()
					}
					cmd.metrics.increase(CommandsSeconds, time.Since(start).Seconds())
				case <-hvmCtx.Done():
					cmdProc.appPartitions = map[istructs.AppQName]*appPartition{} // clear appPartitions to test recovery
					return
				}
			}
		})
	}
}
