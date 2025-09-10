/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type commandProcessorState struct {
	*hostState
	commandPrepareArgs state.CommandPrepareArgsFunc
}

func (s commandProcessorState) CommandPrepareArgs() istructs.CommandPrepareArgs {
	return s.commandPrepareArgs()
}

func implProvideCommandProcessorState(
	ctx context.Context,
	appStructsFunc state.AppStructsFunc,
	partitionIDFunc state.PartitionIDFunc,
	wsidFunc state.WSIDFunc,
	secretReader isecrets.ISecretReader,
	cudFunc state.CUDFunc,
	principalsFunc state.PrincipalsFunc,
	tokenFunc state.TokenFunc,
	intentsLimit int,
	cmdResultBuilderFunc state.ObjectBuilderFunc,
	execCmdArgsFunc state.CommandPrepareArgsFunc,
	argFunc state.ArgFunc,
	unloggedArgFunc state.UnloggedArgFunc,
	wlogOffsetFunc state.WLogOffsetFunc,
	stateCfg state.StateConfig) state.IHostState {

	state := &commandProcessorState{
		hostState:          newHostState("CommandProcessor", intentsLimit, appStructsFunc),
		commandPrepareArgs: execCmdArgsFunc,
	}

	ieventsFunc := func() istructs.IEvents {
		return appStructsFunc().Events()
	}

	state.addStorage(sys.Storage_View, storages.NewViewRecordsStorage(ctx, appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	state.addStorage(sys.Storage_Record, storages.NewRecordsStorage(appStructsFunc, wsidFunc, cudFunc), S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)
	state.addStorage(sys.Storage_WLog, storages.NewWLogStorage(ctx, ieventsFunc, wsidFunc), S_GET)
	state.addStorage(sys.Storage_AppSecret, storages.NewAppSecretsStorage(secretReader), S_GET)
	state.addStorage(sys.Storage_RequestSubject, storages.NewSubjectStorage(principalsFunc, tokenFunc), S_GET)
	state.addStorage(sys.Storage_Result, storages.NewResultStorage(cmdResultBuilderFunc), S_INSERT)
	state.addStorage(sys.Storage_Uniq, storages.NewUniquesStorage(appStructsFunc, wsidFunc, stateCfg.UniquesHandler), S_GET)
	state.addStorage(sys.Storage_Response, storages.NewResponseStorage(), S_INSERT)
	state.addStorage(sys.Storage_CommandContext, storages.NewCommandContextStorage(argFunc, unloggedArgFunc, wsidFunc, wlogOffsetFunc), S_GET)
	state.addStorage(sys.Storage_Logger, storages.NewLoggerStorage(), S_INSERT)

	return state
}
