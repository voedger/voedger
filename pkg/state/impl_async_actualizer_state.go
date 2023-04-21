/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
	"github.com/voedger/voedger/pkg/state/smtptest"
)

type ActualizerStateOptFunc func(opts *actualizerStateOpts)

func WithEmailMessagesChan(messages chan smtptest.Message) ActualizerStateOptFunc {
	return func(opts *actualizerStateOpts) {
		opts.messages = messages
	}
}

type actualizerStateOpts struct {
	messages chan smtptest.Message
}

func implProvideAsyncActualizerState(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, intentsLimit, bundlesLimit int,
	optFuncs ...ActualizerStateOptFunc) IBundledHostState {

	opts := &actualizerStateOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	state := &bundledHostState{
		hostState:    newHostState("AsyncActualizer", intentsLimit),
		bundlesLimit: bundlesLimit,
		bundles:      make(map[schemas.QName]bundle),
	}

	state.addStorage(ViewRecordsStorage, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructs.ViewRecords() },
		schemaCacheFunc: func() schemas.SchemaCache { return appStructs.Schemas() },
		wsidFunc:        wsidFunc,
		n10nFunc:        n10nFunc,
	}, S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)

	state.addStorage(RecordsStorage, &recordsStorage{
		recordsFunc:     func() istructs.IRecords { return appStructs.Records() },
		schemaCacheFunc: func() schemas.SchemaCache { return appStructs.Schemas() },
		wsidFunc:        wsidFunc,
	}, S_GET_BATCH)

	state.addStorage(WLogStorage, &wLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		schemaCacheFunc: func() schemas.SchemaCache { return appStructs.Schemas() },
		wsidFunc:        wsidFunc,
	}, S_GET_BATCH|S_READ)

	state.addStorage(PLogStorage, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		schemaCacheFunc: func() schemas.SchemaCache { return appStructs.Schemas() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET_BATCH|S_READ)

	state.addStorage(SendMailStorage, &sendMailStorage{
		messages: opts.messages,
	}, S_INSERT)

	state.addStorage(HTTPStorage, &httpStorage{}, S_READ)

	state.addStorage(AppSecretsStorage, &appSecretsStorage{secretReader: secretReader}, S_GET_BATCH)

	return state
}
