/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/utils/federation"
)

type QPStateOptFunc func(opts *qpStateOpts)

type qpStateOpts struct {
	customHttpClient         IHttpClient
	federationCommandHandler FederationCommandHandler
}

func QPWithCustomHttpClient(client IHttpClient) QPStateOptFunc {
	return func(opts *qpStateOpts) {
		opts.customHttpClient = client
	}
}

func QPWithFedearationCommandHandler(handler FederationCommandHandler) QPStateOptFunc {
	return func(opts *qpStateOpts) {
		opts.federationCommandHandler = handler
	}
}

func implProvideQueryProcessorState(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc,
	secretReader isecrets.ISecretReader, principalsFunc PrincipalsFunc, tokenFunc TokenFunc, argFunc ArgFunc, resultBuilderFunc ObjectBuilderFunc,
	tokensFunc itokens.ITokens, federationFunc federation.IFederation, queryCallbackFunc ExecQueryCallbackFunc, options ...QPStateOptFunc) IHostState {

	opts := &qpStateOpts{}
	for _, optFunc := range options {
		optFunc(opts)
	}

	bs := newHostState("QueryProcessor", queryProcessorStateMaxIntents, appStructsFunc)

	bs.addStorage(View, newViewRecordsStorage(ctx, appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH|S_READ)
	bs.addStorage(Record, newRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)

	bs.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructsFunc().Events() },
		wsidFunc:   wsidFunc,
	}, S_GET|S_READ)

	bs.addStorage(Http, &httpStorage{
		customClient: opts.customHttpClient,
	}, S_READ)

	bs.addStorage(FederationCommand, &federationCommandStorage{
		appStructs: appStructsFunc,
		wsid:       wsidFunc,
		emulation:  opts.federationCommandHandler,
		federation: federationFunc,
		tokensAPI:  tokensFunc,
	}, S_GET)

	bs.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)

	bs.addStorage(RequestSubject, &subjectStorage{
		principalsFunc: principalsFunc,
		tokenFunc:      tokenFunc,
	}, S_GET)

	bs.addStorage(QueryContext, &queryContextStorage{
		argFunc:  argFunc,
		wsidFunc: wsidFunc,
	}, S_GET)

	bs.addStorage(Response, &cmdResponseStorage{}, S_INSERT)

	bs.addStorage(Result, newQueryResultStorage(appStructsFunc, resultBuilderFunc, queryCallbackFunc), S_INSERT)

	return bs
}
