/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

func implProvideSyncActualizerState(ctx context.Context, appStructs istructs.IAppStructs, partitionIDFunc PartitionIDFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, intentsLimit int) IHostState {
	hs := newHostState("SyncActualizer", intentsLimit)
	hs.addStorage(ViewRecordsStorage, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructs.ViewRecords() },
		schemasFunc:     func() istructs.ISchemas { return appStructs.Schemas() },
		wsidFunc:        wsidFunc,
		n10nFunc:        n10nFunc,
	}, S_GET_BATCH|S_INSERT|S_UPDATE)
	hs.addStorage(RecordsStorage, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructs.Records() },
		schemasFunc: func() istructs.ISchemas { return appStructs.Schemas() },
		wsidFunc:    wsidFunc,
	}, S_GET_BATCH)
	hs.addStorage(WLogStorage, &wLogStorage{
		ctx:         ctx,
		eventsFunc:  func() istructs.IEvents { return appStructs.Events() },
		schemasFunc: func() istructs.ISchemas { return appStructs.Schemas() },
		wsidFunc:    wsidFunc,
	}, S_GET_BATCH)
	hs.addStorage(PLogStorage, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		schemasFunc:     func() istructs.ISchemas { return appStructs.Schemas() },
		partitionIDFunc: partitionIDFunc,
	}, S_GET_BATCH)
	hs.addStorage(AppSecretsStorage, &appSecretsStorage{secretReader: secretReader}, S_GET_BATCH)
	return hs
}
