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

func implProvideCommandProcessorState(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc,
	wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, cudFunc CUDFunc, principalsFunc PrincipalsFunc,
	tokenFunc TokenFunc, intentsLimit int, cmdResultBuilderFunc CmdResultBuilderFunc, argFunc ArgFunc) IHostState {
	bs := newHostState("CommandProcessor", intentsLimit, appStructsFunc)

	bs.addStorage(View, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructsFunc().ViewRecords() },
		appDefFunc:      func() appdef.IAppDef { return appStructsFunc().AppDef() },
		wsidFunc:        wsidFunc,
	}, S_GET|S_GET_BATCH)

	bs.addStorage(Record, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructsFunc().Records() },
		cudFunc:     cudFunc,
		appDefFunc:  func() appdef.IAppDef { return appStructsFunc().AppDef() },
		wsidFunc:    wsidFunc,
	}, S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)

	bs.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructsFunc().Events() },
		appDefFunc: func() appdef.IAppDef { return appStructsFunc().AppDef() },
		wsidFunc:   wsidFunc,
	}, S_GET)

	bs.addStorage(PLog, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructsFunc().Events() },
		appDefFunc:      func() appdef.IAppDef { return appStructsFunc().AppDef() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET)

	bs.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)

	bs.addStorage(RequestSubject, &subjectStorage{
		principalsFunc: principalsFunc,
		tokenFunc:      tokenFunc,
	}, S_GET)

	bs.addStorage(Result, &cmdResultStorage{
		cmdResultBuilderFunc: cmdResultBuilderFunc,
	}, S_INSERT)

	bs.addStorage(Arg, &argStorage{
		argFunc: argFunc,
	}, S_GET)

	return bs
}
