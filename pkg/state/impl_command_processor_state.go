/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/untillpro/voedger/pkg/isecrets"
	"github.com/untillpro/voedger/pkg/istructs"
)

func implProvideCommandProcessorState(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc,
	wsidFunc WSIDFunc, secretReader isecrets.ISecretReader, cudFunc CUDFunc, principalsFunc PrincipalsFunc,
	tokenFunc TokenFunc, intentsLimit int) IHostState {
	bs := newHostState("CommandProcessor", intentsLimit)

	bs.addStorage(ViewRecordsStorage, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructsFunc().ViewRecords() },
		schemasFunc:     func() istructs.ISchemas { return appStructsFunc().Schemas() },
		wsidFunc:        wsidFunc,
	}, S_GET_BATCH)

	bs.addStorage(RecordsStorage, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructsFunc().Records() },
		cudFunc:     cudFunc,
		schemasFunc: func() istructs.ISchemas { return appStructsFunc().Schemas() },
		wsidFunc:    wsidFunc,
	}, S_GET_BATCH|S_INSERT|S_UPDATE)

	bs.addStorage(WLogStorage, &wLogStorage{
		ctx:         ctx,
		eventsFunc:  func() istructs.IEvents { return appStructsFunc().Events() },
		schemasFunc: func() istructs.ISchemas { return appStructsFunc().Schemas() },
		wsidFunc:    wsidFunc,
	}, S_GET_BATCH)

	bs.addStorage(PLogStorage, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructsFunc().Events() },
		schemasFunc:     func() istructs.ISchemas { return appStructsFunc().Schemas() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET_BATCH)

	bs.addStorage(AppSecretsStorage, &appSecretsStorage{secretReader: secretReader}, S_GET_BATCH)

	bs.addStorage(SubjectStorage, &subjectStorage{
		principalsFunc: principalsFunc,
		tokenFunc:      tokenFunc,
	}, S_GET_BATCH)

	return bs
}
