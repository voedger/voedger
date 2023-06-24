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
	tokenFunc TokenFunc, intentsLimit int, cmdResultBuilder istructs.IObjectBuilder) IHostState {
	bs := newHostState("CommandProcessor", intentsLimit)

	bs.addStorage(ViewRecordsStorage, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructsFunc().ViewRecords() },
		appDefFunc:      func() appdef.IAppDef { return appStructsFunc().AppDef() },
		wsidFunc:        wsidFunc,
	}, S_GET_BATCH)

	bs.addStorage(RecordsStorage, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructsFunc().Records() },
		cudFunc:     cudFunc,
		appDefFunc:  func() appdef.IAppDef { return appStructsFunc().AppDef() },
		wsidFunc:    wsidFunc,
	}, S_GET_BATCH|S_INSERT|S_UPDATE)

	bs.addStorage(WLogStorage, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructsFunc().Events() },
		appDefFunc: func() appdef.IAppDef { return appStructsFunc().AppDef() },
		wsidFunc:   wsidFunc,
	}, S_GET_BATCH)

	bs.addStorage(PLogStorage, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructsFunc().Events() },
		appDefFunc:      func() appdef.IAppDef { return appStructsFunc().AppDef() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET_BATCH)

	bs.addStorage(AppSecretsStorage, &appSecretsStorage{secretReader: secretReader}, S_GET_BATCH)

	bs.addStorage(SubjectStorage, &subjectStorage{
		principalsFunc: principalsFunc,
		tokenFunc:      tokenFunc,
	}, S_GET_BATCH)

	bs.addStorage(CmdResultStorage, &cmdResultStorage{cmdResultBuilder: cmdResultBuilder}, S_READ|S_INSERT)

	return bs
}
