/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

func implProvideQueryProcessorState(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc,
	secretReader isecrets.ISecretReader, principalsFunc PrincipalsFunc, tokenFunc TokenFunc, argFunc ArgFunc) IHostState {
	bs := newHostState("QueryProcessor", 0, func() istructs.IAppStructs { return appStructs })

	bs.addStorage(View, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructs.ViewRecords() },
		wsidFunc:        wsidFunc,
	}, S_GET|S_GET_BATCH|S_READ)

	bs.addStorage(Record, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructs.Records() },
		wsidFunc:    wsidFunc,
	}, S_GET|S_GET_BATCH)

	bs.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructs.Events() },
		wsidFunc:   wsidFunc,
	}, S_GET|S_READ)

	bs.addStorage(PLog, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET|S_READ)

	bs.addStorage(Http, &httpStorage{}, S_READ)

	bs.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)

	bs.addStorage(RequestSubject, &subjectStorage{
		principalsFunc: principalsFunc,
		tokenFunc:      tokenFunc,
	}, S_GET)

	return bs
}
