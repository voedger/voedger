/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

// Package testutils provides checkpoint storage doubles shared by recovery tests.
package testutils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/istructs"
)

type OperationKind uint8

const (
	GetPartition OperationKind = iota
	PutPartition
	GetWorkspace
	PutWorkspace
)

// Operation includes the full storage key and, for writes, its value.
type Operation struct {
	Kind        OperationKind
	AppID       istructs.ClusterAppID
	PartitionID istructs.PartitionID
	WSID        istructs.WSID
	Partition   checkpoints.PartitionCheckpoint
	Workspace   checkpoints.WorkspaceCheckpoint
}

type operationKey struct {
	kind        OperationKind
	appID       istructs.ClusterAppID
	partitionID istructs.PartitionID
	wsid        istructs.WSID
}

func (op Operation) key() operationKey {
	if op.Kind == GetPartition || op.Kind == PutPartition {
		return operationKey{kind: op.Kind, appID: op.AppID, partitionID: op.PartitionID}
	}
	return operationKey{kind: op.Kind, appID: op.AppID, wsid: op.WSID}
}

// CheckpointStorage decorates either memory storage or a real backend. Only successful
// writes enter the history. Configuration and snapshots are synchronized;
// callbacks run outside its locks so they can inspect storage or gate recovery.
type CheckpointStorage struct {
	base   checkpoints.IRecoveryCheckpointStorage
	mu     sync.Mutex
	errors map[operationKey]error
	before map[OperationKind]func(Operation) error
	after  map[OperationKind]func(Operation)
	writes []Operation
}

var _ checkpoints.IRecoveryCheckpointStorage = (*CheckpointStorage)(nil)

func NewCheckpointStorageMem() *CheckpointStorage {
	return WrapStorage(&memoryStorage{
		partitions: make(map[operationKey]checkpoints.PartitionCheckpoint),
		workspaces: make(map[operationKey]checkpoints.WorkspaceCheckpoint),
	})
}

func WrapStorage(base checkpoints.IRecoveryCheckpointStorage) *CheckpointStorage {
	return &CheckpointStorage{
		base:   base,
		errors: make(map[operationKey]error),
		before: make(map[OperationKind]func(Operation) error),
		after:  make(map[OperationKind]func(Operation)),
	}
}

// SetError sets a persistent fault for one operation and full key; nil clears it.
func (s *CheckpointStorage) SetError(op Operation, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		delete(s.errors, op.key())
	} else {
		s.errors[op.key()] = err
	}
}

func (s *CheckpointStorage) SetBefore(kind OperationKind, hook func(Operation) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.before[kind] = hook
}

func (s *CheckpointStorage) SetAfter(kind OperationKind, hook func(Operation)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.after[kind] = hook
}

func (s *CheckpointStorage) run(op Operation, call func() error) error {
	s.mu.Lock()
	err, before, after := s.errors[op.key()], s.before[op.Kind], s.after[op.Kind]
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if before != nil {
		if err := before(op); err != nil {
			return err
		}
	}
	if err := call(); err != nil {
		return err
	}
	if op.Kind == PutPartition || op.Kind == PutWorkspace {
		s.mu.Lock()
		s.writes = append(s.writes, op)
		s.mu.Unlock()
	}
	if after != nil {
		after(op)
	}
	return nil
}

func (s *CheckpointStorage) GetPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (value checkpoints.PartitionCheckpoint, found bool, err error) {
	err = s.run(Operation{Kind: GetPartition, AppID: appID, PartitionID: partitionID}, func() error {
		var readErr error
		value, found, readErr = s.base.GetPartitionCheckpoint(appID, partitionID)
		return readErr
	})
	return value, found, err
}

func (s *CheckpointStorage) PutPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID, value checkpoints.PartitionCheckpoint) error {
	return s.run(Operation{Kind: PutPartition, AppID: appID, PartitionID: partitionID, Partition: value}, func() error {
		return s.base.PutPartitionCheckpoint(appID, partitionID, value)
	})
}

func (s *CheckpointStorage) GetWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID) (value checkpoints.WorkspaceCheckpoint, found bool, err error) {
	err = s.run(Operation{Kind: GetWorkspace, AppID: appID, WSID: wsid}, func() error {
		var readErr error
		value, found, readErr = s.base.GetWorkspaceCheckpoint(appID, wsid)
		return readErr
	})
	return value, found, err
}

func (s *CheckpointStorage) PutWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID, value checkpoints.WorkspaceCheckpoint) error {
	return s.run(Operation{Kind: PutWorkspace, AppID: appID, WSID: wsid, Workspace: value}, func() error {
		return s.base.PutWorkspaceCheckpoint(appID, wsid, value)
	})
}

// Force methods bypass faults, hooks and history so fixtures are not counted as recovery writes.
func (s *CheckpointStorage) ForcePutPartitionCheckpoint(t testing.TB, appID istructs.ClusterAppID, partitionID istructs.PartitionID, value checkpoints.PartitionCheckpoint) {
	t.Helper()
	require.NoError(t, s.base.PutPartitionCheckpoint(appID, partitionID, value))
}

func (s *CheckpointStorage) ForcePutWorkspaceCheckpoint(t testing.TB, appID istructs.ClusterAppID, wsid istructs.WSID, value checkpoints.WorkspaceCheckpoint) {
	t.Helper()
	require.NoError(t, s.base.PutWorkspaceCheckpoint(appID, wsid, value))
}

func (s *CheckpointStorage) SuccessfulWrites() []Operation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Operation(nil), s.writes...)
}

func (s *CheckpointStorage) WriteCount(op Operation) int {
	count := 0
	for _, actual := range s.SuccessfulWrites() {
		if actual.key() == op.key() {
			count++
		}
	}
	return count
}

func (s *CheckpointStorage) RequireWriteOrder(t testing.TB, before, after Operation) {
	t.Helper()
	beforeIndex, afterIndex := -1, -1
	for i, op := range s.SuccessfulWrites() {
		if op.key() == before.key() && beforeIndex < 0 {
			beforeIndex = i
		}
		if op.key() == after.key() && afterIndex < 0 {
			afterIndex = i
		}
	}
	require.NotEqual(t, -1, beforeIndex, "earlier write was not made: %+v", before)
	require.NotEqual(t, -1, afterIndex, "later write was not made: %+v", after)
	require.Less(t, beforeIndex, afterIndex)
}

func (s *CheckpointStorage) RequirePartitionCheckpoint(t testing.TB, appID istructs.ClusterAppID, partitionID istructs.PartitionID) checkpoints.PartitionCheckpoint {
	t.Helper()
	value, found, err := s.GetPartitionCheckpoint(appID, partitionID)
	require.NoError(t, err)
	require.True(t, found, "partition %d checkpoint is absent", partitionID)
	return value
}

func (s *CheckpointStorage) RequireWorkspaceCheckpoint(t testing.TB, appID istructs.ClusterAppID, wsid istructs.WSID) checkpoints.WorkspaceCheckpoint {
	t.Helper()
	value, found, err := s.GetWorkspaceCheckpoint(appID, wsid)
	require.NoError(t, err)
	require.True(t, found, "workspace %d checkpoint is absent", wsid)
	return value
}

type memoryStorage struct {
	mu         sync.Mutex
	partitions map[operationKey]checkpoints.PartitionCheckpoint
	workspaces map[operationKey]checkpoints.WorkspaceCheckpoint
}

func (s *memoryStorage) GetPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (checkpoints.PartitionCheckpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found := s.partitions[operationKey{appID: appID, partitionID: partitionID}]
	return value, found, nil
}

func (s *memoryStorage) PutPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID, value checkpoints.PartitionCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.partitions[operationKey{appID: appID, partitionID: partitionID}] = value
	return nil
}

func (s *memoryStorage) GetWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID) (checkpoints.WorkspaceCheckpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found := s.workspaces[operationKey{appID: appID, wsid: wsid}]
	return value, found, nil
}

func (s *memoryStorage) PutWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID, value checkpoints.WorkspaceCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspaces[operationKey{appID: appID, wsid: wsid}] = value
	return nil
}
