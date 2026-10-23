/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/checkpoints/testutils"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

const testClusterAppID istructs.ClusterAppID = 7

func TestInitialRecoveryComputesMissingCheckpoints(t *testing.T) {
	t.Run("missing checkpoints are persisted before marking partitions complete", func(t *testing.T) {
		// Component coverage for checkpoint values and write order; client-visible
		// recovery behavior is verified by TestRecoveryFromCheckpoints in pkg/sys/it.
		appQName := istructs.AppQName_test1_app1
		appID := testClusterAppID
		plog := newMockPLog(t)

		const partitionCount = 3

		// Partition 1 has no checkpoints and two workspace histories. Only the
		// event at WLog offset 6 allocates a record ID.
		plog.setEvents(1, nil,
			newMockPLogEvent(10, 10, 6, newMockCUDRow(istructs.FirstUserRecordID, true)),
			newMockPLogEvent(40, 10, 10),
			newMockPLogEvent(50, 11, 4),
		)
		// Partition 2 has no checkpoints or history in either PLog or WLogs.
		plog.setEvents(2, nil)

		// Partition 0 already has a completion marker.
		storage := testutils.NewCheckpointStorageMem()
		storage.ForcePutPartitionCheckpoint(t, appID, 0, checkpoints.PartitionCheckpoint{LastPLogOffset: 20})

		recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(appID, plog.events), storage)

		deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), appQName, partitionCount)
		require.NoError(t, err)
		checkInitialRecoveryResult(t, partitionCount, deployable, failed)
		require.ElementsMatch(t, []istructs.PartitionID{0, 1, 2}, deployable)
		require.Empty(t, failed)

		// Completed partitions retain their checkpoint without being scanned or rewritten.
		checkpoint, ok, err := storage.GetPartitionCheckpoint(appID, 0)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, checkpoints.PartitionCheckpoint{LastPLogOffset: 20}, checkpoint)
		require.Equal(t, 0, storage.WriteCount(testutils.Operation{Kind: testutils.PutPartition, AppID: testClusterAppID, PartitionID: 0}), "completed partition must not be rewritten")
		require.Equal(t, 0, plog.readPLogCallsCount(0), "completed partition must not be scanned")

		// Missing partition checkpoints are initialized to the actual PLog ends.
		require.Equal(t, checkpoints.PartitionCheckpoint{LastPLogOffset: 50}, storage.RequirePartitionCheckpoint(t, appID, 1))
		require.Equal(t, checkpoints.PartitionCheckpoint{LastPLogOffset: 0}, storage.RequirePartitionCheckpoint(t, appID, 2))

		// Workspace checkpoints point to their latest ID-bearing events, or zero
		// when no such event exists, and precede the partition completion marker.
		expected := checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 6}
		actual := storage.RequireWorkspaceCheckpoint(t, appID, 10)
		require.Equal(t, expected, actual)

		// wsid 11
		expected = checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 0}
		actual = storage.RequireWorkspaceCheckpoint(t, appID, 11)
		require.Equal(t, expected, actual)

		storage.RequireWriteOrder(t,
			testutils.Operation{Kind: testutils.PutWorkspace, AppID: appID, WSID: 10},
			testutils.Operation{Kind: testutils.PutPartition, AppID: appID, PartitionID: 1})
		storage.RequireWriteOrder(t,
			testutils.Operation{Kind: testutils.PutWorkspace, AppID: appID, WSID: 11},
			testutils.Operation{Kind: testutils.PutPartition, AppID: appID, PartitionID: 1})
		require.Equal(t, 1, plog.readPLogCallsCount(1))
		require.Equal(t, 1, plog.readPLogCallsCount(2), "an empty PLog is scanned without allocating log storage")
	})
}

func TestInitialRecoveryUsesTheLatestIDBearingEvent(t *testing.T) {
	appID := testClusterAppID
	plog := newMockPLog(t)

	// Set partition 0 to contain 4 PLog events:
	// workspace 10 creates new records at WLog offsets 6 and 10, then has a CUD-free event at offset 14;
	// workspace 11 only updates an existing record at WLog offset 9.
	plog.setEvents(0, nil,
		newMockPLogEvent(1, 10, 6, newMockCUDRow(istructs.FirstUserRecordID, true)),
		newMockPLogEvent(2, 10, 10, newMockCUDRow(istructs.FirstUserRecordID+1, true)),
		newMockPLogEvent(3, 10, 14),
		newMockPLogEvent(4, 11, 9, newMockCUDRow(istructs.FirstUserRecordID, false)),
	)
	storage := testutils.NewCheckpointStorageMem()
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(appID, plog.events), storage)

	// Run the recovery.
	deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 1)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 1, deployable, failed)
	require.Equal(t, []istructs.PartitionID{0}, deployable)

	// Workspace 10: checkpoint must be on offset 10 since it is the latest ID-bearing event;
	// the later CUD-free event on offset 14 must not advance it.
	expected := checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 10}
	actual := storage.RequireWorkspaceCheckpoint(t, appID, 10)
	require.Equal(t, expected, actual)

	// Workspace 11: checkpoint must be on offset 0 since updating an existing record does not bear a new ID.
	expected = checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 0}
	actual = storage.RequireWorkspaceCheckpoint(t, appID, 11)
	require.Equal(t, expected, actual)

	// Partition 0: checkpoint must be on offset 4 since it is the last PLog event, whether ID-bearing or not.
	expectedPartCP := checkpoints.PartitionCheckpoint{LastPLogOffset: 4}
	actualPartCP := storage.RequirePartitionCheckpoint(t, appID, 0)
	require.Equal(t, expectedPartCP, actualPartCP)
}

func TestInitialRecoveryContinuesAfterPartitionFailures(t *testing.T) {
	injectedErr := errors.New("injected failure")
	tests := []struct {
		name             string
		operation        string
		configureStorage func(*testutils.CheckpointStorage)
		readError        error
		skipRead         bool
		workspaceData    bool
	}{
		{
			name:      "get partition checkpoint",
			operation: "get partition checkpoint",
			skipRead:  true,
			configureStorage: func(storage *testutils.CheckpointStorage) {
				storage.SetError(testutils.Operation{Kind: testutils.GetPartition, AppID: testClusterAppID, PartitionID: 0}, injectedErr)
			},
		},
		{
			name:      "read PLog",
			operation: "read PLog",
			readError: injectedErr,
		},
		{
			name:          "put workspace checkpoint",
			operation:     "put workspace checkpoint",
			workspaceData: true,
			configureStorage: func(storage *testutils.CheckpointStorage) {
				storage.SetError(testutils.Operation{Kind: testutils.PutWorkspace, AppID: testClusterAppID, WSID: 10}, injectedErr)
			},
		},
		{
			name:      "put partition checkpoint",
			operation: "put partition checkpoint",
			configureStorage: func(storage *testutils.CheckpointStorage) {
				storage.SetError(testutils.Operation{Kind: testutils.PutPartition, AppID: testClusterAppID, PartitionID: 0}, injectedErr)
			},
		},
	}

	appID := testClusterAppID

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set partition 0 to fail at the selected recovery stage and partition 1 to recover successfully.
			reads := newMockPLog(t)
			reads.setEvents(1, nil, newMockPLogEvent(3, 20, 2))
			if !test.skipRead {
				entries := []*coreutils.MockPLogEvent(nil)
				if test.workspaceData {
					entries = append(entries, newMockPLogEvent(2, 10, 1))
				}
				reads.setEvents(0, test.readError, entries...)
			}
			storage := testutils.NewCheckpointStorageMem()
			if test.configureStorage != nil {
				test.configureStorage(storage)
			}
			recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(appID, reads.events), storage)

			// Run initial recovery.
			deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 2)
			require.NoError(t, err)
			checkInitialRecoveryResult(t, 2, deployable, failed)
			require.Equal(t, []istructs.PartitionID{1}, deployable)
			require.Len(t, failed, 1)
			require.ErrorIs(t, failed[0], injectedErr)
			require.ErrorContains(t, failed[0], test.operation+": injected failure")

			// Check that the failed partition remains incomplete while the successful partition is checkpointed.
			storage.SetError(testutils.Operation{Kind: testutils.GetPartition, AppID: appID, PartitionID: 0}, nil)
			_, failedPartitionMarked, err := storage.GetPartitionCheckpoint(appID, 0)
			require.NoError(t, err)
			require.False(t, failedPartitionMarked)
			require.Equal(t, checkpoints.PartitionCheckpoint{LastPLogOffset: 3}, storage.RequirePartitionCheckpoint(t, appID, 1))

			// The initializer returns errors for bootstrap to log; no log assertion belongs here.
			require.Zero(t, storage.WriteCount(testutils.Operation{Kind: testutils.PutPartition, AppID: testClusterAppID, PartitionID: 0}))
		})
	}
}

func checkInitialRecoveryResult(t *testing.T, numPartitions int, deployable []istructs.PartitionID,
	failed map[istructs.PartitionID]error) {
	t.Helper()
	seen := map[istructs.PartitionID]bool{}
	for _, partitionID := range deployable {
		require.Less(t, int(partitionID), numPartitions)
		require.False(t, seen[partitionID], "duplicate deployable partition %d", partitionID)
		seen[partitionID] = true
		_, excluded := failed[partitionID]
		require.False(t, excluded, "partition %d is both deployable and failed", partitionID)
	}
	for partitionID, err := range failed {
		require.Less(t, int(partitionID), numPartitions)
		require.Error(t, err)
		seen[partitionID] = true
	}
	require.Len(t, seen, numPartitions, "every declared partition must have exactly one outcome")
}

func TestInitialRecoveryRetriesOnlyIncompletePartitions(t *testing.T) {
	// Set partition 0 to fail during its first PLog read.
	appID := testClusterAppID
	plog := newMockPLog(t)
	plogEvent := newMockPLogEvent(5, 0, 0)
	plog.setEvents(0, errors.New("temporary read failure"), plogEvent)
	checkpointStorage := testutils.NewCheckpointStorageMem()
	checkpointStorage.ForcePutPartitionCheckpoint(t, appID, 1, checkpoints.PartitionCheckpoint{})
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(appID, plog.events), checkpointStorage)

	// Run initial recovery and check that the failedParts partition remains incomplete.
	deployableParts, failedParts, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 2)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 2, deployableParts, failedParts)
	require.Equal(t, []istructs.PartitionID{1}, deployableParts)
	require.Len(t, failedParts, 1)
	_, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, 0)
	require.NoError(t, err)
	require.False(t, ok)

	// Make the PLog read succeed and retry initial recovery.
	plog.setEvents(0, nil, plogEvent)
	deployableParts, failedParts, err = recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 2)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 2, deployableParts, failedParts)
	require.ElementsMatch(t, []istructs.PartitionID{0, 1}, deployableParts)

	// Check that the retried partition is checkpointed after the second read.
	require.Equal(t, checkpoints.PartitionCheckpoint{LastPLogOffset: 5}, checkpointStorage.RequirePartitionCheckpoint(t, appID, 0))
	require.Equal(t, 2, plog.readPLogCallsCount(0))

	// Run initial recovery again and check that the completed partition is skipped.
	deployableParts, failedParts, err = recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 2)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 2, deployableParts, failedParts)
	require.ElementsMatch(t, []istructs.PartitionID{0, 1}, deployableParts)
	require.Equal(t, 2, plog.readPLogCallsCount(0), "completed partition must not be scanned again")
	require.Equal(t, 1, checkpointStorage.WriteCount(testutils.Operation{Kind: testutils.PutPartition, AppID: testClusterAppID, PartitionID: 0}), "completed partition must not be written again")
	require.Zero(t, plog.readPLogCallsCount(1), "previously completed sibling must never be scanned")
	require.Zero(t, checkpointStorage.WriteCount(testutils.Operation{Kind: testutils.PutPartition, AppID: testClusterAppID, PartitionID: 1}))
}

func TestInitialRecoveryHonorsCancellationBeforeScheduling(t *testing.T) {
	appID := testClusterAppID
	plog := newMockPLog(t)
	checkpointStorage := testutils.NewCheckpointStorageMem()
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(appID, plog.events), checkpointStorage)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	deployable, failed, err := recovery.InitializeCheckpoints(ctx, istructs.AppQName_test1_app1, 3)

	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, deployable)
	require.Empty(t, failed, "cancellation is fatal, not a partition failure")
	require.Zero(t, plog.readPLogCallsCountTotal())
	require.Empty(t, checkpointStorage.SuccessfulWrites())
}

func TestInitialRecoveryHasNoRuntimeRecoveryOrPublicationSideEffects(t *testing.T) {
	appID := testClusterAppID
	plog := newMockPLog(t)
	plog.setEvents(0, nil, newMockPLogEvent(1, 10, 1))
	CheckpointStorage := testutils.NewCheckpointStorageMem()
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(appID, plog.events), CheckpointStorage)

	deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 1)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 1, deployable, failed)

	require.Equal(t, []testutils.Operation{
		{Kind: testutils.PutWorkspace, AppID: appID, WSID: 10},
		{Kind: testutils.PutPartition, AppID: appID, Partition: checkpoints.PartitionCheckpoint{LastPLogOffset: 1}},
	}, CheckpointStorage.SuccessfulWrites())
	require.Equal(t, 1, plog.readPLogCallsCount(0))
	plog.events.AssertNotCalled(t, "ReadWLog", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestInitialRecoveryAcceptsPresentZeroCheckpoints(t *testing.T) {
	// Presence, not a nonzero offset, marks completion; no log plog or writes are needed.
	plog := newMockPLog(t)
	checkpointStorage := testutils.NewCheckpointStorageMem()
	for partitionID := range istructs.PartitionID(3) {
		checkpointStorage.ForcePutPartitionCheckpoint(t, testClusterAppID, partitionID, checkpoints.PartitionCheckpoint{})
	}
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(testClusterAppID, plog.events), checkpointStorage)
	deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 3)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 3, deployable, failed)
	require.ElementsMatch(t, []istructs.PartitionID{0, 1, 2}, deployable)
	require.Zero(t, plog.readPLogCallsCountTotal())
	require.Empty(t, checkpointStorage.SuccessfulWrites())
}

func TestInitialRecoveryCheckpointReadFailureExcludesPresentMarker(t *testing.T) {
	// A marker that cannot be read must not authorize deployment, even if it exists in storage.
	plog := newMockPLog(t)
	checkpointStorage := testutils.NewCheckpointStorageMem()
	checkpointStorage.ForcePutPartitionCheckpoint(t, testClusterAppID, 0, checkpoints.PartitionCheckpoint{LastPLogOffset: 42})
	injectedErr := errors.New("checkpoint unavailable")
	checkpointStorage.SetError(testutils.Operation{Kind: testutils.GetPartition, AppID: testClusterAppID, PartitionID: 0}, injectedErr)
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(testClusterAppID, plog.events), checkpointStorage)
	deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 1)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 1, deployable, failed)
	require.Empty(t, deployable)
	require.ErrorIs(t, failed[0], injectedErr)
	require.Zero(t, plog.readPLogCallsCountTotal())
	require.Empty(t, checkpointStorage.SuccessfulWrites())
}

func TestInitialRecoveryReturnsEveryPartitionFailure(t *testing.T) {
	// A complete failure map and an empty deployable slice are a successful, nonfatal invocation.
	plog := newMockPLog(t)
	storage := testutils.NewCheckpointStorageMem()
	injectedErr := errors.New("PLog unavailable")
	for partitionID := range istructs.PartitionID(3) {
		plog.setEvents(partitionID, injectedErr)
	}
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(testClusterAppID, plog.events), storage)
	deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 3)
	require.NoError(t, err)
	checkInitialRecoveryResult(t, 3, deployable, failed)
	require.Empty(t, deployable)
	for partitionID := range istructs.PartitionID(3) {
		require.ErrorIs(t, failed[partitionID], injectedErr)
		require.ErrorContains(t, failed[partitionID], "read PLog")
		require.Equal(t, 1, plog.readPLogCallsCount(partitionID))
	}
	require.Empty(t, storage.SuccessfulWrites())
}

func TestInitialRecoveryReturnsFatalSetupFailure(t *testing.T) {
	setupErr := errors.New("application structures unavailable")
	provider := &coreutils.MockAppStructsProvider{}
	provider.On("BuiltIn", istructs.AppQName_test1_app1).Return(nil, setupErr).Once()
	storage := testutils.NewCheckpointStorageMem()
	recovery := checkpoints.NewInitialRecovery(provider, storage)
	deployable, failed, err := recovery.InitializeCheckpoints(t.Context(), istructs.AppQName_test1_app1, 3)
	require.ErrorIs(t, err, setupErr)
	require.Empty(t, deployable)
	require.Empty(t, failed)
	require.Empty(t, storage.SuccessfulWrites())
	provider.AssertExpectations(t)
}

func TestInitialRecoveryJoinsAttemptsOnCancellation(t *testing.T) {
	// Partition 0 finishes before partition 1 is canceled; partial progress is not a successful outcome.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	completed := make(chan struct{})
	started := make(chan struct{})
	storage := testutils.NewCheckpointStorageMem()
	storage.SetAfter(testutils.PutPartition, func(op testutils.Operation) {
		if op.PartitionID == 0 {
			close(completed)
		}
	})
	reads := newMockPLog(t)
	reads.setEvents(0, nil)
	reads.events.On("ReadPLog", mock.Anything, istructs.PartitionID(1), istructs.FirstOffset,
		istructs.ReadToTheEnd, mock.Anything).Run(func(mock.Arguments) {
		select {
		case <-completed:
		case <-ctx.Done():
			return
		}
		close(started)
		<-ctx.Done()
	}).Return(context.Canceled).Once()
	recovery := checkpoints.NewInitialRecovery(newMockAppStructsProvider(testClusterAppID, reads.events), storage)
	done := make(chan error, 1)
	go func() {
		_, _, err := recovery.InitializeCheckpoints(ctx, istructs.AppQName_test1_app1, 2)
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("partition 1 did not start after partition 0 completed")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.Equal(t, checkpoints.PartitionCheckpoint{}, storage.RequirePartitionCheckpoint(t, testClusterAppID, 0))
	_, ok, err := storage.GetPartitionCheckpoint(testClusterAppID, 1)
	require.NoError(t, err)
	require.False(t, ok)
}

func newMockAppStructsProvider(appID istructs.ClusterAppID, events istructs.IEvents) *coreutils.MockAppStructsProvider {
	asp := &coreutils.MockAppStructsProvider{}
	appStructs := &coreutils.MockAppStructs{}
	appStructs.On("ClusterAppID").Return(appID)
	appStructs.On("Events").Return(events)
	asp.On("BuiltIn", mock.Anything).Return(appStructs, nil)
	return asp
}

func newMockPLogEvent(plogOffset istructs.Offset, wsid istructs.WSID, wLogOffset istructs.Offset, cuds ...istructs.ICUDRow) *coreutils.MockPLogEvent {
	event := &coreutils.MockPLogEvent{}
	event.On("PLogOffset").Return(plogOffset)
	event.On("Workspace").Return(wsid)
	event.On("WLogOffset").Return(wLogOffset)
	event.On("ArgumentObject").Return(istructs.NewNullObject())
	event.On("Synced").Return(false).Maybe()
	event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
		callback := args.Get(0).(func(istructs.ICUDRow) bool)
		for _, cud := range cuds {
			if !callback(cud) {
				return
			}
		}
	}).Return()
	event.On("Release").Return()
	return event
}

func newMockCUDRow(id istructs.RecordID, isNew bool) istructs.ICUDRow {
	row := &coreutils.MockCUDRow{}
	row.On("ID").Return(id)
	row.On("IsNew").Return(isNew)
	return row
}

type mockPLog struct {
	events            *coreutils.MockEvents
	mu                sync.Mutex
	readPLogCallCount map[istructs.PartitionID]int
}

func newMockPLog(t *testing.T) *mockPLog {
	t.Helper()
	recorder := &mockPLog{
		events:            &coreutils.MockEvents{},
		readPLogCallCount: map[istructs.PartitionID]int{},
	}
	t.Cleanup(func() { recorder.events.AssertExpectations(t) })
	return recorder
}

func (r *mockPLog) setEvents(partitionID istructs.PartitionID, errOnReadPLog error,
	events ...*coreutils.MockPLogEvent) {
	expectation := r.events.On("ReadPLog", mock.Anything, partitionID, istructs.FirstOffset,
		istructs.ReadToTheEnd, mock.Anything).Return(errOnReadPLog).Once()
	expectation.Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		callback := args.Get(4).(istructs.PLogEventsReaderCallback)
		r.onReadPLog(partitionID)
		if err := ctx.Err(); err != nil {
			expectation.Return(err)
			return
		}
		if errOnReadPLog != nil {
			return
		}
		for _, event := range events {
			if err := callback(event.PLogOffset(), event); err != nil {
				expectation.Return(err)
				return
			}
		}
		expectation.Return(nil)
	})
}

func (r *mockPLog) onReadPLog(partitionID istructs.PartitionID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.readPLogCallCount[partitionID]++
}

func (r *mockPLog) readPLogCallsCount(partitionID istructs.PartitionID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readPLogCallCount[partitionID]
}

func (r *mockPLog) readPLogCallsCountTotal() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, count := range r.readPLogCallCount {
		total += count
	}
	return total
}
