/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ttlstorage

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func TestInsertIfNotExist(t *testing.T) {
	mockVVMAppTTLStorage := new(MockIVVMAppTTLStorage)
	const (
		pKeyPrefix        PKeyPrefix        = 0xABCD1234
		ttlStorageImplKey TTLStorageImplKey = 12345
		val                                 = "some-value"
	)
	ttlStorage := New(pKeyPrefix, mockVVMAppTTLStorage)

	// Preallocate and check slices for non-empty
	pKey := uint32ToBytes(pKeyPrefix)
	require.Len(t, pKey, 4, "pKey should have length 4")

	cCols := uint32ToBytes(ttlStorageImplKey)
	require.Len(t, cCols, 4, "cCols should have length 4")

	ttlSeconds := 5

	mockVVMAppTTLStorage.On("InsertIfNotExists", pKey, cCols, []byte(val), ttlSeconds).
		Return(true, nil)

	ok, err := ttlStorage.InsertIfNotExist(ttlStorageImplKey, val, time.Duration(ttlSeconds)*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	mockVVMAppTTLStorage.AssertExpectations(t)
}

func TestCompareAndSwap(t *testing.T) {
	mockVVMAppTTLStorage := new(MockIVVMAppTTLStorage)
	const (
		pKeyPrefix        PKeyPrefix        = 0xABCD1234
		ttlStorageImplKey TTLStorageImplKey = 12345
		oldVal                              = "old-val"
		newVal                              = "new-val"
	)
	ttlStorage := New(pKeyPrefix, mockVVMAppTTLStorage)

	pKey := uint32ToBytes(pKeyPrefix)
	require.Len(t, pKey, 4)
	cCols := uint32ToBytes(ttlStorageImplKey)
	require.Len(t, cCols, 4)

	ttlSeconds := 10

	mockVVMAppTTLStorage.On("CompareAndSwap", pKey, cCols, []byte(oldVal), []byte(newVal), ttlSeconds).
		Return(true, nil)

	ok, err := ttlStorage.CompareAndSwap(ttlStorageImplKey, oldVal, newVal, time.Duration(ttlSeconds)*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	mockVVMAppTTLStorage.AssertExpectations(t)
}

func TestCompareAndDelete(t *testing.T) {
	mockVVMAppTTLStorage := new(MockIVVMAppTTLStorage)
	const (
		pKeyPrefix        PKeyPrefix        = 0xABCD1234
		ttlStorageImplKey TTLStorageImplKey = 12345
		val                                 = "val"
	)
	ttlStorage := New(pKeyPrefix, mockVVMAppTTLStorage)

	pKey := uint32ToBytes(pKeyPrefix)
	require.Len(t, pKey, 4)
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
