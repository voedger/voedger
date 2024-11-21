/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package teststate

import (
	"github.com/voedger/voedger/pkg/istructs"
	"time"
)

type IFullQName interface {
	PkgPath() string
	Entity() string
}

type IView interface {
	IFullQName
	Keys() []string
}

type ICommandRunner interface {
	// methos to fulfill test state
	StateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner
	StateSingletonRecord(fQName IFullQName, keyValueList ...any) ICommandRunner
	ArgumentObject(id istructs.RecordID, keyValueList ...any) ICommandRunner
	ArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) ICommandRunner
	// methods to check out the test state
	IntentSingletonInsert(fQName IFullQName, keyValueList ...any) ICommandRunner
	IntentSingletonUpdate(fQName IFullQName, keyValueList ...any) ICommandRunner
	IntentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner
	IntentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) ICommandRunner
	// method to run the test
	Run()
}

type IProjectorRunner interface {
	// State related methods
	StateRecord(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	StateSingletonRecord(fQName IFullQName, keyValueList ...any) IProjectorRunner
	StateCUDRow(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	StateView(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	// Event related methods
	EventWLogOffset(offset istructs.Offset) IProjectorRunner
	EventPLogOffset(offset istructs.Offset) IProjectorRunner
	EventRegisteredAt(registeredAt time.Time) IProjectorRunner
	EventSyncedAt(syncedAt time.Time) IProjectorRunner
	EventCUD(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	EventQName(qName IFullQName) IProjectorRunner
	EventSynced(synced bool) IProjectorRunner
	EventDeviceID(deviceID istructs.ConnectedDeviceID) IProjectorRunner
	EventWSID(wsid istructs.WSID) IProjectorRunner
	EventArgumentObject(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	EventArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	EventUnloggedArgumentObject(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	EventUnloggedArgumentObjectRow(path string, id istructs.RecordID, keyValueList ...any) IProjectorRunner

	// Intent related methods for checking out the test state
	IntentSingletonInsert(fQName IFullQName, keyValueList ...any) IProjectorRunner
	IntentSingletonUpdate(fQName IFullQName, keyValueList ...any) IProjectorRunner
	IntentRecordInsert(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	IntentRecordUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	IntentViewInsert(fQName IFullQName, keyValueList ...any) IProjectorRunner
	IntentViewUpdate(fQName IFullQName, id istructs.RecordID, keyValueList ...any) IProjectorRunner
	// Run is a method for running test
	Run()
}

type ICommand interface {
	IFullQName
	ArgumentPkgPath() string
	ArgumentEntity() string
	WorkspaceDescriptor() string
}

type IProjector interface {
	IFullQName
	WorkspaceDescriptor() string
}
