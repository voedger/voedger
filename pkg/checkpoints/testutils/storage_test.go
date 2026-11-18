/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package testutils

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestCheckpointStorageSeparatesKeysAndReplacesValues(t *testing.T) {
	require := require.New(t)
	storage := NewCheckpointStorageMem()
	_, found, err := storage.GetPartitionCheckpoint(1, 1)
	require.NoError(err)
	require.False(found)
	_, found, err = storage.GetWorkspaceCheckpoint(1, 1)
	require.NoError(err)
	require.False(found)

	// Application, identifier, and checkpoint kind must all isolate stored values
	for _, appID := range []istructs.ClusterAppID{1, 2} {
		for id := range 2 {
			offset := istructs.Offset(int(appID)*100 + id)
			require.NoError(storage.PutPartitionCheckpoint(appID, istructs.PartitionID(id), checkpoints.PartitionCheckpoint{LastPLogOffset: offset}))
			require.NoError(storage.PutWorkspaceCheckpoint(appID, istructs.WSID(id), checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: offset + 10}))
		}
	}
	// Replacement is unconditional: zero overwrites a larger offset and remains present
	require.NoError(storage.PutPartitionCheckpoint(1, 1, checkpoints.PartitionCheckpoint{}))
	require.NoError(storage.PutWorkspaceCheckpoint(1, 1, checkpoints.WorkspaceCheckpoint{}))
	for _, appID := range []istructs.ClusterAppID{1, 2} {
		for id := range 2 {
			expectedPartition := checkpoints.PartitionCheckpoint{LastPLogOffset: istructs.Offset(int(appID)*100 + id)}
			expectedWorkspace := checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: expectedPartition.LastPLogOffset + 10}
			if appID == 1 && id == 1 {
				expectedPartition = checkpoints.PartitionCheckpoint{}
				expectedWorkspace = checkpoints.WorkspaceCheckpoint{}
			}
			actualPartition, found, err := storage.GetPartitionCheckpoint(appID, istructs.PartitionID(id))
			require.NoError(err)
			require.True(found)
			require.Equal(expectedPartition, actualPartition)
			actualWorkspace, found, err := storage.GetWorkspaceCheckpoint(appID, istructs.WSID(id))
			require.NoError(err)
			require.True(found)
			require.Equal(expectedWorkspace, actualWorkspace)
		}
	}
}

func TestCheckpointStorageFailuresDoNotCommitOrRunAfterHooks(t *testing.T) {
	require := require.New(t)
	for _, test := range []struct {
		name    string
		op      Operation
		otherID Operation
	}{
		{"get partition", Operation{Kind: GetPartition, AppID: 1, PartitionID: 1}, Operation{Kind: GetPartition, AppID: 1, PartitionID: 2}},
		{"put partition", Operation{Kind: PutPartition, AppID: 1, PartitionID: 1, Partition: checkpoints.PartitionCheckpoint{LastPLogOffset: 20}}, Operation{Kind: PutPartition, AppID: 1, PartitionID: 2}},
		{"get workspace", Operation{Kind: GetWorkspace, AppID: 1, WSID: 1}, Operation{Kind: GetWorkspace, AppID: 1, WSID: 2}},
		{"put workspace", Operation{Kind: PutWorkspace, AppID: 1, WSID: 1, Workspace: checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 21}}, Operation{Kind: PutWorkspace, AppID: 1, WSID: 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := NewCheckpointStorageMem()
			storage := WrapStorage(base)
			base.ForcePutPartitionCheckpoint(t, 1, 1, checkpoints.PartitionCheckpoint{LastPLogOffset: 10})
			base.ForcePutWorkspaceCheckpoint(t, 1, 1, checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 11})
			op := test.op
			// Faults apply only to the full key, and remain active until cleared
			expectedError := errors.New("injected failure")
			storage.SetError(op, expectedError)
			var expectedWrites []Operation
			otherApp := op
			otherApp.AppID = 2
			for _, other := range []Operation{otherApp, test.otherID} {
				require.NoError(callCheckpointStorage(storage, other))
				if other.Kind == PutPartition || other.Kind == PutWorkspace {
					expectedWrites = append(expectedWrites, other)
				}
			}
			beforeCalls, afterCalls := 0, 0
			beforeError := errors.New("before hook failure")
			storage.SetBefore(op.Kind, func(Operation) error { beforeCalls++; return beforeError })
			storage.SetAfter(op.Kind, func(Operation) { afterCalls++ })
			for range 2 {
				require.ErrorIs(callCheckpointStorage(storage, op), expectedError)
			}
			require.Zero(beforeCalls)
			storage.SetError(op, nil)
			require.ErrorIs(callCheckpointStorage(storage, op), beforeError)
			require.Equal(1, beforeCalls)
			require.Zero(afterCalls)
			require.Equal(expectedWrites, storage.SuccessfulWrites())

			// Backend errors also prevent recording and after hooks, without changing data
			storage.SetBefore(op.Kind, nil)
			backendError := errors.New("backend failure")
			base.SetError(op, backendError)
			require.ErrorIs(callCheckpointStorage(storage, op), backendError)
			require.Zero(afterCalls)
			require.Equal(expectedWrites, storage.SuccessfulWrites())
			base.SetError(op, nil)
			require.Equal(checkpoints.PartitionCheckpoint{LastPLogOffset: 10}, base.RequirePartitionCheckpoint(t, 1, 1))
			require.Equal(checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 11}, base.RequireWorkspaceCheckpoint(t, 1, 1))

			require.NoError(callCheckpointStorage(storage, op))
			require.Equal(1, afterCalls)
			switch op.Kind {
			case PutPartition:
				require.Equal(op.Partition, base.RequirePartitionCheckpoint(t, 1, 1))
			case PutWorkspace:
				require.Equal(op.Workspace, base.RequireWorkspaceCheckpoint(t, 1, 1))
			}
		})
	}
}

func TestCheckpointStorageObservesOrdinaryWritesButNotFixtureSetup(t *testing.T) {
	require := require.New(t)
	base := NewCheckpointStorageMem()
	storage := WrapStorage(base)
	workspaceWrite := Operation{Kind: PutWorkspace, AppID: 1, WSID: 3, Workspace: checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 12}}
	partitionWrite := Operation{Kind: PutPartition, AppID: 1, PartitionID: 2, Partition: checkpoints.PartitionCheckpoint{LastPLogOffset: 40}}
	hookCalls := 0
	for _, op := range []Operation{workspaceWrite, partitionWrite} {
		storage.SetError(op, errors.New("write blocked"))
		storage.SetBefore(op.Kind, func(Operation) error { hookCalls++; return errors.New("hook blocked") })
		storage.SetAfter(op.Kind, func(Operation) { hookCalls++ })
	}
	// Fixture writes must reach the backend without invoking faults, hooks, or history
	fixturePartition := checkpoints.PartitionCheckpoint{LastPLogOffset: 30}
	fixtureWorkspace := checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 10}
	storage.ForcePutPartitionCheckpoint(t, 1, 2, fixturePartition)
	storage.ForcePutWorkspaceCheckpoint(t, 1, 3, fixtureWorkspace)
	require.Equal(fixturePartition, base.RequirePartitionCheckpoint(t, 1, 2))
	require.Equal(fixtureWorkspace, base.RequireWorkspaceCheckpoint(t, 1, 3))
	require.Zero(hookCalls)
	require.Empty(storage.SuccessfulWrites())

	for _, op := range []Operation{workspaceWrite, partitionWrite} {
		storage.SetError(op, nil)
		storage.SetBefore(op.Kind, nil)
		storage.SetAfter(op.Kind, nil)
	}
	var before, after struct {
		op         Operation
		checkpoint checkpoints.WorkspaceCheckpoint
		found      bool
		err        error
		writes     []Operation
	}
	storage.SetBefore(PutWorkspace, func(op Operation) error {
		before.op = op
		before.checkpoint, before.found, before.err = storage.GetWorkspaceCheckpoint(op.AppID, op.WSID)
		before.writes = storage.SuccessfulWrites()
		storage.SetBefore(PutWorkspace, nil)
		return nil
	})
	storage.SetAfter(PutWorkspace, func(op Operation) {
		after.op = op
		after.checkpoint, after.found, after.err = storage.GetWorkspaceCheckpoint(op.AppID, op.WSID)
		after.writes = storage.SuccessfulWrites()
		storage.SetAfter(PutWorkspace, nil)
	})
	// Reentrant hooks must see the old and committed values without deadlocking
	done := make(chan error, 1)
	go func() { done <- callCheckpointStorage(storage, workspaceWrite) }()
	select {
	case err := <-done:
		require.NoError(err)
	case <-time.After(5 * time.Second):
		require.FailNow("storage hooks blocked while reentering storage")
	}
	require.Equal(workspaceWrite, before.op)
	require.NoError(before.err)
	require.True(before.found)
	require.Equal(fixtureWorkspace, before.checkpoint)
	require.Empty(before.writes)
	require.Equal(workspaceWrite, after.op)
	require.NoError(after.err)
	require.True(after.found)
	require.Equal(workspaceWrite.Workspace, after.checkpoint)
	require.Equal([]Operation{workspaceWrite}, after.writes)

	require.NoError(callCheckpointStorage(storage, partitionWrite))
	expectedWrites := []Operation{workspaceWrite, partitionWrite}
	actualWrites := storage.SuccessfulWrites()
	require.Equal(expectedWrites, actualWrites)
	// Mutating a returned snapshot must not corrupt subsequent history observations
	actualWrites[0].WSID = 99
	require.Equal(expectedWrites, storage.SuccessfulWrites())
}

func TestCheckpointStorageSupportsConcurrentAccess(t *testing.T) {
	require := require.New(t)
	storage := NewCheckpointStorageMem()
	const numWorkers, numWrites = 16, 16
	actualErrors := make(chan error, numWorkers)
	var workers sync.WaitGroup
	for worker := range numWorkers {
		workers.Go(func() {
			appID := istructs.ClusterAppID(worker%2 + 1)
			partitionID := istructs.PartitionID(worker/2 + 1)
			wsid := istructs.WSID(partitionID)
			for iteration := range numWrites {
				storage.SetError(Operation{Kind: GetPartition, AppID: appID, PartitionID: partitionID}, nil)
				storage.SetBefore(GetPartition, nil)
				storage.SetAfter(GetWorkspace, nil)
				offset := istructs.Offset(worker*numWrites + iteration + 1)
				for _, kind := range []OperationKind{PutPartition, PutWorkspace, GetPartition, GetWorkspace} {
					op := Operation{Kind: kind, AppID: appID, PartitionID: partitionID, WSID: wsid,
						Partition: checkpoints.PartitionCheckpoint{LastPLogOffset: offset},
						Workspace: checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: offset}}
					if err := callCheckpointStorage(storage, op); err != nil {
						actualErrors <- err
						return
					}
				}
				storage.SuccessfulWrites()
			}
		})
	}
	workers.Wait()
	close(actualErrors)
	for err := range actualErrors {
		require.NoError(err)
	}
	// Every key retains its own final write and every successful write is recorded once
	require.Len(storage.SuccessfulWrites(), 2*numWorkers*numWrites)
	for worker := range numWorkers {
		appID := istructs.ClusterAppID(worker%2 + 1)
		partitionID := istructs.PartitionID(worker/2 + 1)
		expected := istructs.Offset((worker + 1) * numWrites)
		require.Equal(checkpoints.PartitionCheckpoint{LastPLogOffset: expected}, storage.RequirePartitionCheckpoint(t, appID, partitionID))
		require.Equal(checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: expected}, storage.RequireWorkspaceCheckpoint(t, appID, istructs.WSID(partitionID)))
		require.Equal(numWrites, storage.WriteCount(Operation{Kind: PutPartition, AppID: appID, PartitionID: partitionID}))
		require.Equal(numWrites, storage.WriteCount(Operation{Kind: PutWorkspace, AppID: appID, WSID: istructs.WSID(partitionID)}))
	}
}

func callCheckpointStorage(storage *CheckpointStorage, op Operation) error {
	switch op.Kind {
	case GetPartition:
		_, _, err := storage.GetPartitionCheckpoint(op.AppID, op.PartitionID)
		return err
	case PutPartition:
		return storage.PutPartitionCheckpoint(op.AppID, op.PartitionID, op.Partition)
	case GetWorkspace:
		_, _, err := storage.GetWorkspaceCheckpoint(op.AppID, op.WSID)
		return err
	case PutWorkspace:
		return storage.PutWorkspaceCheckpoint(op.AppID, op.WSID, op.Workspace)
	default:
		panic("invalid checkpoint operation")
	}
}
