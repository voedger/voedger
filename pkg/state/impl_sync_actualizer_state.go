/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

func implProvideSyncActualizerState(ctx context.Context, appStructs istructs.IAppStructs, iWorkspaceFunc iWorkspaceFunc, partitionIDFunc PartitionIDFunc,
	wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, intentsLimit int) IHostState {
	hs := newHostState("SyncActualizer", intentsLimit)
	hs.addStorage(View, &viewRecordsStorage{
		ctx:             ctx,
		viewRecordsFunc: func() istructs.IViewRecords { return appStructs.ViewRecords() },
		iWorkspaceFunc:  iWorkspaceFunc,
		wsidFunc:        wsidFunc,
		n10nFunc:        n10nFunc,
	}, S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)
	hs.addStorage(Record, &recordsStorage{
		recordsFunc:    func() istructs.IRecords { return appStructs.Records() },
		iWorkspaceFunc: iWorkspaceFunc,
		wsidFunc:       wsidFunc,
	}, S_GET|S_GET_BATCH)
	hs.addStorage(WLog, &wLogStorage{
		ctx:            ctx,
		eventsFunc:     func() istructs.IEvents { return appStructs.Events() },
		iWorkspaceFunc: iWorkspaceFunc,
		wsidFunc:       wsidFunc,
	}, S_GET)
	hs.addStorage(PLog, &pLogStorage{
		ctx:             ctx,
		eventsFunc:      func() istructs.IEvents { return appStructs.Events() },
		iWorkspaceFunc:  iWorkspaceFunc,
		partitionIDFunc: partitionIDFunc,
	}, S_GET)
	hs.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)
	return hs
}
