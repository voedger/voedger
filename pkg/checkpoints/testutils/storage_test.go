/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package testutils

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestMemoryStorage(t *testing.T) {
	storage := NewCheckpointStorageMem()
	partition, found, err := storage.GetPartitionCheckpoint(1, 2)
	require.NoError(t, err)
	require.False(t, found)
	require.Zero(t, partition)
	workspace, found, err := storage.GetWorkspaceCheckpoint(1, 2)
	require.NoError(t, err)
	require.False(t, found)
	require.Zero(t, workspace)

	// Present zero values differ from missing checkpoints.
	storage.ForcePutPartitionCheckpoint(t, 1, 2, checkpoints.PartitionCheckpoint{})
	storage.ForcePutWorkspaceCheckpoint(t, 1, 2, checkpoints.WorkspaceCheckpoint{})
	require.Zero(t, storage.RequirePartitionCheckpoint(t, 1, 2))
	require.Zero(t, storage.RequireWorkspaceCheckpoint(t, 1, 2))
	require.Empty(t, storage.SuccessfulWrites())

	for _, key := range [][2]uint64{{1, 2}, {1, 3}, {2, 2}} {
		appID, partitionID, wsid := istructs.ClusterAppID(key[0]), istructs.PartitionID(key[1]), istructs.WSID(key[1])
		offset := istructs.Offset(key[0]*10 + key[1])
		require.NoError(t, storage.PutPartitionCheckpoint(appID, partitionID, checkpoints.PartitionCheckpoint{LastPLogOffset: offset}))
		require.NoError(t, storage.PutWorkspaceCheckpoint(appID, wsid, checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: offset}))
	}
	for _, key := range [][2]uint64{{1, 2}, {1, 3}, {2, 2}} {
		offset := istructs.Offset(key[0]*10 + key[1])
		require.Equal(t, offset, storage.RequirePartitionCheckpoint(t, istructs.ClusterAppID(key[0]), istructs.PartitionID(key[1])).LastPLogOffset)
		require.Equal(t, offset, storage.RequireWorkspaceCheckpoint(t, istructs.ClusterAppID(key[0]), istructs.WSID(key[1])).WLogIDBearingOffset)
	}

	// A later, smaller value wins; this storage must not keep a maximum.
	require.NoError(t, storage.PutPartitionCheckpoint(1, 2, checkpoints.PartitionCheckpoint{LastPLogOffset: 1}))
	require.NoError(t, storage.PutWorkspaceCheckpoint(1, 2, checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 1}))
	require.Equal(t, istructs.Offset(1), storage.RequirePartitionCheckpoint(t, 1, 2).LastPLogOffset)
	require.Equal(t, istructs.Offset(1), storage.RequireWorkspaceCheckpoint(t, 1, 2).WLogIDBearingOffset)
}

func TestStorageFaultsAndSeeding(t *testing.T) {
	for _, kind := range []OperationKind{GetPartition, PutPartition, GetWorkspace, PutWorkspace} {
		t.Run(operationName(kind), func(t *testing.T) {
			storage := NewCheckpointStorageMem()
			op := Operation{Kind: kind, AppID: 1, PartitionID: 2, WSID: 2}
			injected := errors.New("injected failure")
			storage.SetError(op, injected)
			calls := 0
			storage.SetBefore(kind, func(Operation) error { calls++; return nil })
			storage.SetAfter(kind, func(Operation) { calls++ })
			storage.ForcePutPartitionCheckpoint(t, 1, 2, checkpoints.PartitionCheckpoint{LastPLogOffset: 4})
			storage.ForcePutWorkspaceCheckpoint(t, 1, 2, checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 5})
			require.Zero(t, calls, "seeding bypasses hooks")
			require.Empty(t, storage.SuccessfulWrites())
			require.ErrorIs(t, callStorage(storage, op), injected)
			require.Zero(t, calls, "injected errors bypass callbacks")
			require.Empty(t, storage.SuccessfulWrites(), "failed writes must not be recorded")

			otherApp := op
			otherApp.AppID = 2
			require.NoError(t, callStorage(storage, otherApp), "faults are isolated by application")
			require.Equal(t, 2, calls)
			storage.SetError(op, nil)
			require.NoError(t, callStorage(storage, op))
			require.Equal(t, 4, calls)
		})
	}
}

func TestStorageWrapsBackend(t *testing.T) {
	backend := NewMockStorage(t)
	storage := WrapStorage(backend)
	value := checkpoints.PartitionCheckpoint{LastPLogOffset: 9}
	injected := errors.New("backend failure")
	backend.On("GetPartitionCheckpoint", istructs.ClusterAppID(1), istructs.PartitionID(2)).
		Return(value, true, nil).Once()
	backend.On("PutPartitionCheckpoint", istructs.ClusterAppID(1), istructs.PartitionID(2), value).
		Return(injected).Once()
	backend.On("PutPartitionCheckpoint", istructs.ClusterAppID(1), istructs.PartitionID(3), value).
		Return(nil).Once()

	actual, found, err := storage.GetPartitionCheckpoint(1, 2)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, value, actual)
	completed := 0
	storage.SetAfter(PutPartition, func(Operation) {
		completed++
		// Reentrant inspection proves that the callback does not hold the storage lock.
		require.Len(t, storage.SuccessfulWrites(), 1)
	})
	require.ErrorIs(t, storage.PutPartitionCheckpoint(1, 2, value), injected)
	require.Empty(t, storage.SuccessfulWrites())
	require.Zero(t, completed)
	require.NoError(t, storage.PutPartitionCheckpoint(1, 3, value))
	require.Equal(t, 1, completed)
	require.Equal(t, []Operation{{Kind: PutPartition, AppID: 1, PartitionID: 3, Partition: value}}, storage.SuccessfulWrites())

	writes := storage.SuccessfulWrites()
	writes[0].Partition.LastPLogOffset = 100
	require.Equal(t, value, storage.SuccessfulWrites()[0].Partition, "snapshots must not expose mutable history")
}

func TestStorageHooksAndWriteOrder(t *testing.T) {
	storage := NewCheckpointStorageMem()
	injected := errors.New("before-hook failure")
	storage.SetBefore(PutPartition, func(Operation) error { return injected })
	require.ErrorIs(t, storage.PutPartitionCheckpoint(1, 2, checkpoints.PartitionCheckpoint{}), injected)
	_, found, err := storage.GetPartitionCheckpoint(1, 2)
	require.NoError(t, err)
	require.False(t, found)
	storage.SetBefore(PutPartition, nil)

	workspaceWrite := Operation{Kind: PutWorkspace, AppID: 1, WSID: 10}
	partitionWrite := Operation{Kind: PutPartition, AppID: 1, PartitionID: 2}
	require.NoError(t, storage.PutWorkspaceCheckpoint(1, 10, checkpoints.WorkspaceCheckpoint{}))
	require.NoError(t, storage.PutPartitionCheckpoint(1, 2, checkpoints.PartitionCheckpoint{}))
	storage.RequireWriteOrder(t, workspaceWrite, partitionWrite)
	require.Equal(t, 1, storage.WriteCount(partitionWrite))
	require.Zero(t, storage.WriteCount(Operation{Kind: PutPartition, AppID: 2, PartitionID: 2}))
}

func TestStorageConcurrentAccess(t *testing.T) {
	storage := NewCheckpointStorageMem()
	var workers sync.WaitGroup
	results := make(chan error, 64)
	for i := range 64 {
		workers.Go(func() {
			appID, partitionID := istructs.ClusterAppID(i/16), istructs.PartitionID(i%16)
			op := Operation{Kind: GetPartition, AppID: appID, PartitionID: partitionID}
			storage.SetError(op, nil)
			storage.SetAfter(PutWorkspace, nil)
			storage.SuccessfulWrites()
			if err := storage.PutPartitionCheckpoint(appID, partitionID, checkpoints.PartitionCheckpoint{LastPLogOffset: 1}); err != nil {
				results <- err
				return
			}
			_, _, err := storage.GetPartitionCheckpoint(appID, partitionID)
			results <- err
		})
	}
	workers.Wait()
	for range 64 {
		require.NoError(t, <-results)
	}
	require.Len(t, storage.SuccessfulWrites(), 64)
}

func TestMockStorageRejectsUnexpectedCalls(t *testing.T) {
	storage := &MockCheckpointStorage{}
	storage.On("PutWorkspaceCheckpoint", istructs.ClusterAppID(1), istructs.WSID(2), checkpoints.WorkspaceCheckpoint{}).
		Return(nil).Once()
	require.Panics(t, func() { _ = storage.PutWorkspaceCheckpoint(1, 3, checkpoints.WorkspaceCheckpoint{}) })
	require.NoError(t, storage.PutWorkspaceCheckpoint(1, 2, checkpoints.WorkspaceCheckpoint{}))
	require.Panics(t, func() { _ = storage.PutWorkspaceCheckpoint(1, 2, checkpoints.WorkspaceCheckpoint{}) })
	storage.AssertExpectations(t)
}

func callStorage(storage *CheckpointStorage, op Operation) error {
	switch op.Kind {
	case GetPartition:
		_, _, err := storage.GetPartitionCheckpoint(op.AppID, op.PartitionID)
		return err
	case PutPartition:
		return storage.PutPartitionCheckpoint(op.AppID, op.PartitionID, op.Partition)
	case GetWorkspace:
		_, _, err := storage.GetWorkspaceCheckpoint(op.AppID, op.WSID)
		return err
	default:
		return storage.PutWorkspaceCheckpoint(op.AppID, op.WSID, op.Workspace)
	}
}

func operationName(kind OperationKind) string {
	return [...]string{"get partition", "put partition", "get workspace", "put workspace"}[kind]
}
