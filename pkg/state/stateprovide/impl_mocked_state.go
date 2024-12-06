/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package stateprovide

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

type MockedState struct {
	*hostState
}

func implProvideMockedCommandProcessorState(intentsLimit int, appStructsFunc state.AppStructsFunc) state.IHostState {

	ms := &MockedState{
		hostState: newHostState("MockedCommandProcessorState", intentsLimit, appStructsFunc),
	}

	ms.addStorage(sys.Storage_View, storages.NewMockedStorage(sys.Storage_View), S_GET|S_GET_BATCH)
	ms.addStorage(sys.Storage_Record, storages.NewMockedStorage(sys.Storage_Record), S_GET|S_GET_BATCH|S_INSERT|S_UPDATE)
	ms.addStorage(sys.Storage_WLog, storages.NewMockedStorage(sys.Storage_WLog), S_GET)
	ms.addStorage(sys.Storage_AppSecret, storages.NewMockedStorage(sys.Storage_AppSecret), S_GET)
	ms.addStorage(sys.Storage_RequestSubject, storages.NewMockedStorage(sys.Storage_RequestSubject), S_GET)
	ms.addStorage(sys.Storage_Result, storages.NewMockedStorage(sys.Storage_Result), S_INSERT)
	ms.addStorage(sys.Storage_Uniq, storages.NewMockedStorage(sys.Storage_Uniq), S_GET)
	ms.addStorage(sys.Storage_Response, storages.NewMockedStorage(sys.Storage_Response), S_INSERT)
	ms.addStorage(sys.Storage_CommandContext, storages.NewMockedStorage(sys.Storage_CommandContext), S_GET)
	ms.addStorage(sys.Storage_Logger, storages.NewMockedStorage(sys.Storage_Logger), S_INSERT)

	return ms
}

func implProvideMockedActualizerState(intentsLimit int, appStructsFunc state.AppStructsFunc) state.IHostState {

	ms := &MockedState{
		hostState: newHostState("MockedActualizerState", intentsLimit, appStructsFunc),
	}

	ms.addStorage(sys.Storage_View, storages.NewMockedStorage(sys.Storage_View), S_GET|S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)
	ms.addStorage(sys.Storage_Record, storages.NewMockedStorage(sys.Storage_Record), S_GET|S_GET_BATCH)
	ms.addStorage(sys.Storage_Event, storages.NewMockedStorage(sys.Storage_Event), S_GET)
	ms.addStorage(sys.Storage_WLog, storages.NewMockedStorage(sys.Storage_WLog), S_GET|S_READ)
	ms.addStorage(sys.Storage_SendMail, storages.NewMockedStorage(sys.Storage_SendMail), S_INSERT)
	ms.addStorage(sys.Storage_Http, storages.NewMockedStorage(sys.Storage_Http), S_READ)
	ms.addStorage(sys.Storage_FederationCommand, storages.NewMockedStorage(sys.Storage_FederationCommand), S_GET)
	ms.addStorage(sys.Storage_FederationBlob, storages.NewMockedStorage(sys.Storage_FederationBlob), S_READ)
	ms.addStorage(sys.Storage_AppSecret, storages.NewMockedStorage(sys.Storage_AppSecret), S_GET)
	ms.addStorage(sys.Storage_Uniq, storages.NewMockedStorage(sys.Storage_Uniq), S_GET)
	ms.addStorage(sys.Storage_Logger, storages.NewMockedStorage(sys.Storage_Logger), S_INSERT)

	return ms
}

func (ms *MockedState) GetMockedStorage(storageName appdef.QName) (*storages.MockedStorage, bool) {
	st, ok := ms.storages[storageName]
	if !ok {
		return nil, false
	}

	return st.(*storages.MockedStorage), ok
}
