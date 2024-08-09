/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type syncActualizerState struct {
	*hostState
	eventFunc PLogEventFunc
}

func (s *syncActualizerState) PLogEvent() istructs.IPLogEvent {
	return s.eventFunc()
}

func implProvideSyncActualizerState(ctx context.Context, appStructsFunc AppStructsFunc, partitionIDFunc PartitionIDFunc,
	wsidFunc WSIDFunc, n10nFunc N10nFunc, secretReader isecrets.ISecretReader, eventFunc PLogEventFunc, intentsLimit int, options ...StateOptFunc) IHostState {
	opts := &stateOpts{}
	for _, optFunc := range options {
		optFunc(opts)
	}
	hs := &syncActualizerState{
		hostState: newHostState("SyncActualizer", intentsLimit, appStructsFunc),
		eventFunc: eventFunc,
	}
	hs.addStorage(sys.Storage_View, newViewRecordsStorage(ctx, appStructsFunc, wsidFunc, n10nFunc), S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)
	hs.addStorage(sys.Storage_Record, newRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	hs.addStorage(sys.Storage_WLog, &wLogStorage{
		ctx:        ctx,
		eventsFunc: func() istructs.IEvents { return appStructsFunc().Events() },
		wsidFunc:   wsidFunc,
	}, S_GET)
	hs.addStorage(sys.Storage_AppSecret, &appSecretsStorage{secretReader: secretReader}, S_GET)
	hs.addStorage(sys.Storage_Uniq, newUniquesStorage(appStructsFunc, wsidFunc, opts.uniquesHandler), S_GET)
	return hs
}
