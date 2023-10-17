/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

func implProvideQueryProcessorState(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc,
	secretReader isecrets.ISecretReader, principalsFunc PrincipalsFunc, tokenFunc TokenFunc) IHostState {
	bs := newHostState("QueryProcessor", 0)

	bs.addStorage(View, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructs.ViewRecords() },
		appDefFunc:      func() appdef.IAppDef { return appStructs.AppDef() },
		wsidFunc:        wsidFunc,
	}, S_GET|S_GET_BATCH|S_READ)

	bs.addStorage(Record, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructs.Records() },
		appDefFunc:  func() appdef.IAppDef { return appStructs.AppDef() },
		wsidFunc:    wsidFunc,
	}, S_GET|S_GET_BATCH)

	bs.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructs.Events() },
		appDefFunc: func() appdef.IAppDef { return appStructs.AppDef() },
		wsidFunc:   wsidFunc,
	}, S_GET|S_READ)

	bs.addStorage(PLog, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		appDefFunc:      func() appdef.IAppDef { return appStructs.AppDef() },
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
