/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
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

func implProvideAsyncActualizerState(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, intentsLimit, bundlesLimit int,
	optFuncs ...ActualizerStateOptFunc) IBundledHostState {

	opts := &actualizerStateOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	state := &bundledHostState{
		hostState:    newHostState("AsyncActualizer", intentsLimit, func() istructs.IAppStructs { return appStructs }),
		bundlesLimit: bundlesLimit,
		bundles:      make(map[appdef.QName]bundle),
	}

	state.addStorage(View, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructs.ViewRecords() },
		wsidFunc:        wsidFunc,
		n10nFunc:        n10nFunc,
	}, S_GET|S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)

	state.addStorage(Record, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructs.Records() },
		wsidFunc:    wsidFunc,
	}, S_GET|S_GET_BATCH)

	state.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructs.Events() },
		wsidFunc:   wsidFunc,
	}, S_GET|S_READ)

	state.addStorage(PLog, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET|S_READ)

	state.addStorage(SendMail, &sendMailStorage{
		messages: opts.messages,
	}, S_INSERT)

	state.addStorage(Http, &httpStorage{}, S_READ)

	state.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)

	state.addStorage(Event, &eventStorage{eventFunc: eventFunc}, S_GET)

	return state
}
