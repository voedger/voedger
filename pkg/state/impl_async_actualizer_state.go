/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/utils/federation"
)

type asyncActualizerState struct {
	*bundledHostState
	eventFunc PLogEventFunc
}

func (s *asyncActualizerState) PLogEvent() istructs.IPLogEvent {
	return s.eventFunc()
}

func implProvideAsyncActualizerState(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc,
	secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, tokensFunc itokens.ITokens, federationFunc federation.IFederation,
	intentsLimit, bundlesLimit int, optFuncs ...StateOptFunc) IBundledHostState {

	opts := &stateOpts{}
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

	state.addStorage(View, newViewRecordsStorage(ctx, appStructsFunc, wsidFunc, n10nFunc), S_GET|S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)
	state.addStorage(Record, newRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	state.addStorage(Event, newEventStorage(eventFunc), S_GET)

	state.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructsFunc().Events() },
		wsidFunc:   wsidFunc,
	}, S_GET|S_READ)

	state.addStorage(SendMail, &sendMailStorage{
		messages: opts.messages,
	}, S_INSERT)

	state.addStorage(Http, &httpStorage{
		customClient: opts.customHttpClient,
	}, S_READ)

	state.addStorage(FederationCommand, &federationCommandStorage{
		appStructs: appStructsFunc,
		wsid:       wsidFunc,
		emulation:  opts.federationCommandHandler,
		federation: federationFunc,
		tokens:     tokensFunc,
	}, S_GET)

	state.addStorage(FederationBlob, &federationBlobStorage{
		appStructs: appStructsFunc,
		wsid:       wsidFunc,
		emulation:  opts.federationBlobHandler,
		federation: federationFunc,
		tokens:     tokensFunc,
	}, S_READ)

	state.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)

	state.addStorage(Event, &eventStorage{eventFunc: eventFunc}, S_GET)

	state.addStorage(Uniq, newUniquesStorage(appStructsFunc, wsidFunc, opts.uniquesHandler), S_GET)

	return state
}
