/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

type syncActualizerState struct {
	*hostState
	eventFunc PLogEventFunc
}

func (s *syncActualizerState) PLogEvent() istructs.IPLogEvent {
	return s.eventFunc()
}

func implProvideSyncActualizerState(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc,
	wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, intentsLimit int) IHostState {
	hs := &syncActualizerState{
		hostState: newHostState("SyncActualizer", intentsLimit, appStructsFunc),
		eventFunc: eventFunc,
	}
	hs.addStorage(View, newViewRecordsStorage(ctx, appStructsFunc, wsidFunc, n10nFunc), S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)
	hs.addStorage(Record, &recordsStorage{
		recordsFunc: func() istructs.IRecords { return appStructsFunc().Records() },
		wsidFunc:    wsidFunc,
	}, S_GET|S_GET_BATCH)
	hs.addStorage(WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructsFunc().Events() },
		wsidFunc:   wsidFunc,
	}, S_GET)
	hs.addStorage(AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)
	return hs
}
