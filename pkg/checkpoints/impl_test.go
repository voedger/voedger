/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/checkpoints/testutils"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func TestApplyRecoveryCheckpointUpdatesPLogCheckpointEachHundredthEvent(t *testing.T) {
	require := require.New(t)
	storage := testutils.NewMockStorage(t)
	projector := checkpoints.ApplyRecoveryCheckpoint(storage)
	state := newIState(t)
	event := newIPLogEvent(t, 99)

	require.NoError(projector(event, state, nil))
	storage.AssertNotCalled(t, "PutPartitionCheckpoint", mock.Anything, mock.Anything, mock.Anything)
	storage.AssertNotCalled(t, "PutWorkspaceCheckpoint", mock.Anything, mock.Anything, mock.Anything)

	event = newIPLogEvent(t, 100)
	event.On("HandlingPartition").Return(istructs.PartitionID(4))
	storage.On("PutPartitionCheckpoint", istructs.ClusterApps[istructs.AppQName_test1_app1],
		istructs.PartitionID(4), checkpoints.PartitionCheckpoint{LastPLogOffset: 100}).Return(nil).Once()
	require.NoError(projector(event, state, nil))
	storage.AssertExpectations(t)

	event = newIPLogEvent(t, 101)
	require.NoError(projector(event, state, nil))
	storage.AssertNumberOfCalls(t, "PutPartitionCheckpoint", 1)

	event = newIPLogEvent(t, 200)
	event.On("HandlingPartition").Return(istructs.PartitionID(4))
	storage.On("PutPartitionCheckpoint", istructs.ClusterApps[istructs.AppQName_test1_app1],
		istructs.PartitionID(4), checkpoints.PartitionCheckpoint{LastPLogOffset: 200}).Return(nil).Once()
	require.NoError(projector(event, state, nil))
}

func TestApplyRecoveryCheckpointWritesOnlyIDBearingWLogOffsets(t *testing.T) {
	require := require.New(t)
	storage := testutils.NewMockStorage(t)
	projector := checkpoints.ApplyRecoveryCheckpoint(storage)
	state := newIState(t)
	singleton := newCheckpointTestCUDRow(t, istructs.FirstSingletonID, true)
	updated := newCheckpointTestCUDRow(t, istructs.FirstUserRecordID, false)
	event := newIPLogEvent(t, 1, singleton, updated)

	require.NoError(projector(event, state, nil))
	storage.AssertNotCalled(t, "PutPartitionCheckpoint", mock.Anything, mock.Anything, mock.Anything)
	storage.AssertNotCalled(t, "PutWorkspaceCheckpoint", mock.Anything, mock.Anything, mock.Anything)

	event = newIPLogEvent(t, 1, singleton, updated,
		newCheckpointTestCUDRow(t, istructs.FirstUserRecordID, true))
	event.On("Workspace").Return(istructs.WSID(5))
	event.On("WLogOffset").Return(istructs.Offset(20))
	storage.On("PutWorkspaceCheckpoint", istructs.ClusterApps[istructs.AppQName_test1_app1],
		istructs.WSID(5), checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 20}).Return(nil).Once()
	require.NoError(projector(event, state, nil))
	storage.AssertExpectations(t)

	event = newIPLogEvent(t, 2)
	require.NoError(projector(event, state, nil))
	storage.AssertNumberOfCalls(t, "PutWorkspaceCheckpoint", 1)

	event = newIPLogEvent(t, 3,
		newCheckpointTestCUDRow(t, istructs.FirstUserRecordID+1, true))
	event.On("Workspace").Return(istructs.WSID(6))
	event.On("WLogOffset").Return(istructs.Offset(22))
	storage.On("PutWorkspaceCheckpoint", istructs.ClusterApps[istructs.AppQName_test1_app1],
		istructs.WSID(6), checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 22}).Return(nil).Once()
	require.NoError(projector(event, state, nil))
}

func TestIDBearingEvent(t *testing.T) {
	t.Run("new ordinary record", func(t *testing.T) {
		event := newIPLogEvent(t, 0,
			newCheckpointTestCUDRow(t, istructs.FirstUserRecordID, true))
		require.True(t, checkpoints.IsIDBearingEvent(event))
	})

	t.Run("new singleton", func(t *testing.T) {
		event := newIPLogEvent(t, 0,
			newCheckpointTestCUDRow(t, istructs.FirstSingletonID, true))
		require.False(t, checkpoints.IsIDBearingEvent(event))
	})

	t.Run("existing ordinary record update", func(t *testing.T) {
		event := newIPLogEvent(t, 0,
			newCheckpointTestCUDRow(t, istructs.FirstUserRecordID, false))
		require.False(t, checkpoints.IsIDBearingEvent(event))
	})

	t.Run("CUD-free event", func(t *testing.T) {
		require.False(t, checkpoints.IsIDBearingEvent(newIPLogEvent(t, 0)))
	})
}

func TestApplyRecoveryCheckpointReturnsStorageErrors(t *testing.T) {
	require := require.New(t)
	partitionErr := errors.New("injected partition checkpoint failure")
	workspaceErr := errors.New("injected workspace checkpoint failure")
	storage := testutils.NewMockStorage(t)
	storage.On("PutPartitionCheckpoint", istructs.ClusterApps[istructs.AppQName_test1_app1],
		istructs.PartitionID(0), checkpoints.PartitionCheckpoint{LastPLogOffset: 100}).Return(partitionErr).Once()
	storage.On("PutWorkspaceCheckpoint", istructs.ClusterApps[istructs.AppQName_test1_app1],
		istructs.WSID(0), checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 10}).Return(workspaceErr).Once()
	state := newIState(t)
	event := newIPLogEvent(t, 100)
	event.On("HandlingPartition").Return(istructs.PartitionID(0))

	require.ErrorIs(checkpoints.ApplyRecoveryCheckpoint(storage)(event, state, nil), partitionErr)

	event = newIPLogEvent(t, 101,
		newCheckpointTestCUDRow(t, istructs.FirstUserRecordID, true))
	event.On("Workspace").Return(istructs.WSID(0))
	event.On("WLogOffset").Return(istructs.Offset(10))
	require.ErrorIs(checkpoints.ApplyRecoveryCheckpoint(storage)(event, state, nil), workspaceErr)
}

func TestProvideRegistersOneStandardAsyncProjector(t *testing.T) {
	require := require.New(t)
	resources := istructsmem.NewStatelessResources()
	storage := testutils.NewMockStorage(t)
	checkpoints.Provide(resources, storage)

	found := map[appdef.QName]bool{}
	resources.Projectors(func(path string, projector istructs.Projector) bool {
		require.Equal(appdef.SysPackagePath, path)
		found[projector.Name] = true
		return true
	})
	require.Equal(map[appdef.QName]bool{checkpoints.QNameApplyRecoveryCheckpoint: true}, found)
}

func newIState(t *testing.T) *coreutils.MockState {
	t.Helper()
	appStructs := &coreutils.MockAppStructs{}
	appStructs.On("ClusterAppID").Return(istructs.ClusterApps[istructs.AppQName_test1_app1])
	state := &coreutils.MockState{}
	state.On("AppStructs").Return(appStructs)
	t.Cleanup(func() {
		state.AssertExpectations(t)
		appStructs.AssertExpectations(t)
	})
	return state
}

func newIPLogEvent(t *testing.T, offset istructs.Offset, cuds ...istructs.ICUDRow) *coreutils.MockPLogEvent {
	t.Helper()
	event := &coreutils.MockPLogEvent{}
	// The predicate reads only CUDs; a failed partition write skips CUDs entirely.
	event.On("PLogOffset").Return(offset).Maybe()
	event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
		callback := args.Get(0).(func(istructs.ICUDRow) bool)
		for _, cud := range cuds {
			if !callback(cud) {
				return
			}
		}
	}).Maybe()
	t.Cleanup(func() { event.AssertExpectations(t) })
	return event
}

func newCheckpointTestCUDRow(t *testing.T, id istructs.RecordID, isNew bool) *coreutils.MockCUDRow {
	t.Helper()
	row := &coreutils.MockCUDRow{}
	row.On("ID").Return(id).Maybe() // Short-circuit evaluation may skip ID for an existing row.
	row.On("IsNew").Return(isNew)
	t.Cleanup(func() { row.AssertExpectations(t) })
	return row
}
