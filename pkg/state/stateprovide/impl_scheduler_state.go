/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package stateprovide

import (
	"context"

	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type schedulerState struct {
	*hostState
}

func implProvideSchedulerState(ctx context.Context, appStructsFunc state.AppStructsFunc, wsidFunc state.WSIDFunc, n10nFunc state.N10nFunc,
	secretReader isecrets.ISecretReader, tokensFunc itokens.ITokens, federationFunc federation.IFederation, unixTimeFunc state.UnixTimeFunc,
	intentsLimit int, stateCfg state.StateConfig) state.IHostState {

	state := &schedulerState{
		hostState: newHostState("Scheduler", intentsLimit, appStructsFunc),
	}

	ieventsFunc := func() istructs.IEvents {
		return appStructsFunc().Events()
	}

	state.addStorage(sys.Storage_View, storages.NewViewRecordsStorage(ctx, appStructsFunc, wsidFunc, n10nFunc), S_GET|S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)
	state.addStorage(sys.Storage_Record, storages.NewRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	state.addStorage(sys.Storage_WLog, storages.NewWLogStorage(ctx, ieventsFunc, wsidFunc), S_GET|S_READ)
	state.addStorage(sys.Storage_SendMail, storages.NewSendMailStorage(stateCfg.MessagesSenderOverride), S_GET|S_INSERT)
	state.addStorage(sys.Storage_HTTP, storages.NewHTTPStorage(stateCfg.CustomHTTPClient), S_READ)
	state.addStorage(sys.Storage_FederationCommand, storages.NewFederationCommandStorage(appStructsFunc, wsidFunc, federationFunc, tokensFunc, stateCfg.FederationCommandHandler), S_GET)
	state.addStorage(sys.Storage_FederationBlob, storages.NewFederationBlobStorage(appStructsFunc, wsidFunc, federationFunc, tokensFunc, stateCfg.FederationBlobHandler), S_READ)
	state.addStorage(sys.Storage_AppSecret, storages.NewAppSecretsStorage(secretReader), S_GET)
	state.addStorage(sys.Storage_Uniq, storages.NewUniquesStorage(appStructsFunc, wsidFunc, stateCfg.UniquesHandler), S_GET)
	state.addStorage(sys.Storage_JobContext, storages.NewJobContextStorage(wsidFunc, unixTimeFunc), S_GET)
	state.addStorage(sys.Storage_Logger, storages.NewLoggerStorage(), S_INSERT)

	return state
}
