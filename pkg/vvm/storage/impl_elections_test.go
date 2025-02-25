/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func TestInsertIfNotExist(t *testing.T) {
	mockVVMAppTTLStorage := new(MockIVVMAppTTLStorage)
	const (
		ttlStorageImplKey TTLStorageImplKey = 12345
		val                                 = "some-value"
	)
	ttlStorage := NewElectionsTTLStorage(mockVVMAppTTLStorage)

	require.Equal(t, pKeyPrefix_VVMLeader, ttlStorage.(*implElectionsITTLStorage).prefix)

	// Preallocate and check slices for non-empty
	pKey := uint32ToBytes(pKeyPrefix_VVMLeader)

	cCols := uint32ToBytes(ttlStorageImplKey)
	require.Len(t, cCols, 4, "cCols should have length 4")

	ttlSeconds := 5

	mockVVMAppTTLStorage.On("InsertIfNotExists", pKey, cCols, []byte(val), ttlSeconds).
		Return(true, nil)

	ok, err := ttlStorage.InsertIfNotExist(ttlStorageImplKey, val, ttlSeconds)
	require.NoError(t, err)
	require.True(t, ok)

	mockVVMAppTTLStorage.AssertExpectations(t)
}

func TestCompareAndSwap(t *testing.T) {
	mockVVMAppTTLStorage := new(MockIVVMAppTTLStorage)
	const (
		ttlStorageImplKey TTLStorageImplKey = 12345
		oldVal                              = "old-val"
		newVal                              = "new-val"
	)
	ttlStorage := NewElectionsTTLStorage(mockVVMAppTTLStorage)

	pKey := uint32ToBytes(pKeyPrefix_VVMLeader)
	cCols := uint32ToBytes(ttlStorageImplKey)
	require.Len(t, cCols, 4)

	ttlSeconds := 10

	mockVVMAppTTLStorage.On("CompareAndSwap", pKey, cCols, []byte(oldVal), []byte(newVal), ttlSeconds).
		Return(true, nil)

	ok, err := ttlStorage.CompareAndSwap(ttlStorageImplKey, oldVal, newVal, ttlSeconds)
	require.NoError(t, err)
	require.True(t, ok)

	mockVVMAppTTLStorage.AssertExpectations(t)
}

func TestCompareAndDelete(t *testing.T) {
	mockVVMAppTTLStorage := new(MockIVVMAppTTLStorage)
	const (
		ttlStorageImplKey TTLStorageImplKey = 12345
		val                                 = "val"
	)
	ttlStorage := NewElectionsTTLStorage(mockVVMAppTTLStorage)

	pKey := uint32ToBytes(pKeyPrefix_VVMLeader)
	cCols := uint32ToBytes(ttlStorageImplKey)
	require.Len(t, cCols, 4)

	mockVVMAppTTLStorage.On("CompareAndDelete", pKey, cCols, []byte(val)).
		Return(true, nil)

	ok, err := ttlStorage.CompareAndDelete(ttlStorageImplKey, val)
	require.NoError(t, err)
	require.True(t, ok)

	mockVVMAppTTLStorage.AssertExpectations(t)
}

func uint32ToBytes(val uint32) []byte {
	res := make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(res, val)
	return res
}

// MockIVVMAppTTLStorage is a mock for the IVVMAppTTLStorage interface
type MockIVVMAppTTLStorage struct {
	mock.Mock
}

func (m *MockIVVMAppTTLStorage) InsertIfNotExists(pKey, cCols, value []byte, ttlSeconds int) (bool, error) {
	args := m.Called(pKey, cCols, value, ttlSeconds)
	return args.Bool(0), args.Error(1)
}

func (m *MockIVVMAppTTLStorage) CompareAndSwap(pKey, cCols, oldValue, newValue []byte, ttlSeconds int) (bool, error) {
	args := m.Called(pKey, cCols, oldValue, newValue, ttlSeconds)
	return args.Bool(0), args.Error(1)
}

func (m *MockIVVMAppTTLStorage) CompareAndDelete(pKey, cCols, expectedValue []byte) (bool, error) {
	args := m.Called(pKey, cCols, expectedValue)
	return args.Bool(0), args.Error(1)
}
