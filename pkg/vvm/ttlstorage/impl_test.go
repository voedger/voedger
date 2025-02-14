/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ttlstorage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func TestInsertIfNotExist(t *testing.T) {
	mockStore := new(MockIVVMAppTTLStorage)
	store := New(0xABCD1234, mockStore)

	// Preallocate and check slices for non-empty
	pKey := []byte{0xAB, 0xCD, 0x12, 0x34}
	require.Len(t, pKey, 4, "pKey should have length 4")
	cCols := []byte{0x00, 0x00, 0x30, 0x39}
	require.Len(t, cCols, 4, "cCols should have length 4")

	val := []byte("some-value")
	ttlSeconds := 5

	mockStore.On("InsertIfNotExists", pKey, cCols, val, ttlSeconds).
		Return(true, nil)

	ok, err := store.InsertIfNotExist(12345, "some-value", 5*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	mockStore.AssertExpectations(t)
}

func TestCompareAndSwap(t *testing.T) {
	mockStore := new(MockIVVMAppTTLStorage)
	store := New(0xABCD1234, mockStore)

	pKey := []byte{0xAB, 0xCD, 0x12, 0x34}
	require.Len(t, pKey, 4)
	cCols := []byte{0x00, 0x00, 0x30, 0x39}
	require.Len(t, cCols, 4)

	oldVal := []byte("old-val")
	newVal := []byte("new-val")
	ttlSeconds := 10

	mockStore.On("CompareAndSwap", pKey, cCols, oldVal, newVal, ttlSeconds).
		Return(true, nil)

	ok, err := store.CompareAndSwap(12345, "old-val", "new-val", 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	mockStore.AssertExpectations(t)
}

func TestCompareAndDelete(t *testing.T) {
	mockStore := new(MockIVVMAppTTLStorage)
	store := New(0xABCD1234, mockStore)

	pKey := []byte{0xAB, 0xCD, 0x12, 0x34}
	require.Len(t, pKey, 4)
	cCols := []byte{0x00, 0x00, 0x30, 0x39}
	require.Len(t, cCols, 4)

	val := []byte("val")

	mockStore.On("CompareAndDelete", pKey, cCols, val).
		Return(true, nil)

	ok, err := store.CompareAndDelete(12345, "val")
	require.NoError(t, err)
	require.True(t, ok)

	mockStore.AssertExpectations(t)
}
