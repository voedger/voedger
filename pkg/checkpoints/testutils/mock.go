/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package testutils

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/istructs"
)

// MockCheckpointStorage is strict: unexpected calls never fall back to another storage.
type MockCheckpointStorage struct {
	mock.Mock
}

var _ checkpoints.IRecoveryCheckpointStorage = (*MockCheckpointStorage)(nil)

func NewMockStorage(t testing.TB) *MockCheckpointStorage {
	t.Helper()
	storage := &MockCheckpointStorage{}
	storage.Test(t)
	t.Cleanup(func() { storage.AssertExpectations(t) })
	return storage
}

func (s *MockCheckpointStorage) GetPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (checkpoints.PartitionCheckpoint, bool, error) {
	args := s.Called(appID, partitionID)
	return args.Get(0).(checkpoints.PartitionCheckpoint), args.Bool(1), args.Error(2)
}

func (s *MockCheckpointStorage) PutPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID, value checkpoints.PartitionCheckpoint) error {
	return s.Called(appID, partitionID, value).Error(0)
}

func (s *MockCheckpointStorage) GetWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID) (checkpoints.WorkspaceCheckpoint, bool, error) {
	args := s.Called(appID, wsid)
	return args.Get(0).(checkpoints.WorkspaceCheckpoint), args.Bool(1), args.Error(2)
}

func (s *MockCheckpointStorage) PutWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID, value checkpoints.WorkspaceCheckpoint) error {
	return s.Called(appID, wsid, value).Error(0)
}
