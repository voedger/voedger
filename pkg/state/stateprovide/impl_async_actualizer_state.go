/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type asyncActualizerState struct {
	*bundledHostState
	eventFunc state.PLogEventFunc
}

func (s *asyncActualizerState) PLogEvent() istructs.IPLogEvent {
	return s.eventFunc()
}

func implProvideAsyncActualizerState(ctx context.Context, appStructsFunc state.AppStructsFunc, partitionIDFunc state.PartitionIDFunc, wsidFunc state.WSIDFunc, n10nFunc state.N10nFunc,
	secretReader isecrets.ISecretReader, eventFunc state.PLogEventFunc, tokensFunc itokens.ITokens, federationFunc federation.IFederation,
	intentsLimit, bundlesLimit int, optFuncs ...state.StateOptFunc) state.IBundledHostState {

	opts := &state.StateOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	state := &asyncActualizerState{
		bundledHostState: &bundledHostState{
			hostState:    newHostState("AsyncActualizer", intentsLimit, appStructsFunc),
			bundlesLimit: bundlesLimit,
			bundles:      make(map[appdef.QName]bundle),
		},
		eventFunc: eventFunc,
	}

	ieventsFunc := func() istructs.IEvents {
		return appStructsFunc().Events()
	}

	state.addStorage(sys.Storage_View, storages.NewViewRecordsStorage(ctx, appStructsFunc, wsidFunc, n10nFunc), S_GET|S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)
	state.addStorage(sys.Storage_Record, storages.NewRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	state.addStorage(sys.Storage_Event, storages.NewEventStorage(eventFunc), S_GET)
	state.addStorage(sys.Storage_WLog, storages.NewWLogStorage(ctx, ieventsFunc, wsidFunc), S_GET|S_READ)
	state.addStorage(sys.Storage_SendMail, storages.NewSendMailStorage(opts.MessagesSenderOverride), S_INSERT)
	state.addStorage(sys.Storage_HTTP, storages.NewHTTPStorage(opts.CustomHTTPClient), S_READ)
	state.addStorage(sys.Storage_FederationCommand, storages.NewFederationCommandStorage(appStructsFunc, wsidFunc, federationFunc, tokensFunc, opts.FederationCommandHandler), S_GET)
	state.addStorage(sys.Storage_FederationBlob, storages.NewFederationBlobStorage(appStructsFunc, wsidFunc, federationFunc, tokensFunc, opts.FederationBlobHandler), S_READ)
	state.addStorage(sys.Storage_AppSecret, storages.NewAppSecretsStorage(secretReader), S_GET)
	state.addStorage(sys.Storage_Uniq, storages.NewUniquesStorage(appStructsFunc, wsidFunc, opts.UniquesHandler), S_GET)
	state.addStorage(sys.Storage_Logger, storages.NewLoggerStorage(), S_INSERT)

	return state
}
