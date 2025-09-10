/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"context"

	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type syncActualizerState struct {
	*hostState
	eventFunc state.PLogEventFunc
}

func (s *syncActualizerState) PLogEvent() istructs.IPLogEvent {
	return s.eventFunc()
}

func implProvideSyncActualizerState(ctx context.Context, appStructsFunc state.AppStructsFunc, partitionIDFunc state.PartitionIDFunc,
	wsidFunc state.WSIDFunc, n10nFunc state.N10nFunc, secretReader isecrets.ISecretReader, eventFunc state.PLogEventFunc, intentsLimit int, stateCfg state.StateConfig) state.IHostState {
	hs := &syncActualizerState{
		hostState: newHostState("SyncActualizer", intentsLimit, appStructsFunc),
		eventFunc: eventFunc,
	}
	ieventsFunc := func() istructs.IEvents {
		return appStructsFunc().Events()
	}
	hs.addStorage(sys.Storage_View, storages.NewViewRecordsStorage(ctx, appStructsFunc, wsidFunc, n10nFunc), S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)
	hs.addStorage(sys.Storage_Record, storages.NewRecordsStorage(appStructsFunc, wsidFunc, nil), S_GET|S_GET_BATCH)
	hs.addStorage(sys.Storage_WLog, storages.NewWLogStorage(ctx, ieventsFunc, wsidFunc), S_GET)
	hs.addStorage(sys.Storage_AppSecret, storages.NewAppSecretsStorage(secretReader), S_GET)
	hs.addStorage(sys.Storage_Uniq, storages.NewUniquesStorage(appStructsFunc, wsidFunc, stateCfg.UniquesHandler), S_GET)
	hs.addStorage(sys.Storage_Logger, storages.NewLoggerStorage(), S_INSERT)
	return hs
}
