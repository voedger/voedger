/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

// this used as pKey. Actual value does not matter in tests
var pKeyPrefix = []byte{1}

const seconds10 = 10

func TestElections_BasicUsage(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()
	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	ctx := elector.AcquireLeadership("testKey", "leaderVal", seconds10)
	require.NotNil(t, ctx, "Should return a non-nil context on successful acquisition")

	storedData := iTTLStorage.data["testKey"]
	require.Equal(t, "leaderVal", storedData.value)

	// Release leadership
	elector.ReleaseLeadership("testKey")
	<-ctx.Done()
}

func TestElections_AcquireLeadership_Failure(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	iTTLStorage.data["occupied"] = valueWithTTL[string]{
		value:     "someVal",
		expiresAt: coreutils.MockTime.Now().Add(seconds10 * time.Second),
	}

	ctx := elector.AcquireLeadership("occupied", "myVal", seconds10)
	require.Nil(t, ctx, "Should return nil if the key is already in storage")
}

func TestElections_AcquireLeadership_Error(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()

	iTTLStorage.errorTrigger["badKey"] = true

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	ctx := elector.AcquireLeadership("badKey", "val", seconds10)
	require.Nil(t, ctx, "Should return nil if a storage error occurs")
}

func TestElections_CompareAndSwap_RenewFails(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	ctx := elector.AcquireLeadership("renewKey", "renewVal", seconds10)
	require.NotNil(t, ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	iTTLStorage.data["renewKey"] = valueWithTTL[string]{
		value:     "someOtherVal",
		expiresAt: coreutils.MockTime.Now().Add(seconds10 * 2 * time.Second),
	}

	// trigger the renewal
	coreutils.MockTime.Sleep((seconds10 + 1) * time.Second)

	// The leadership is forcibly released in the background.
	<-ctx.Done()
	require.ErrorIs(t, ctx.Err(), context.Canceled, "Context should be canceled after renewal failure")

	keptValue := iTTLStorage.data["renewKey"]
	require.Equal(t, "someOtherVal", keptValue.value)
}

func TestElections_ReleaseLeadership_NoLeader(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	// Releasing unknown key => no effect
	elector.ReleaseLeadership("unknownKey")
}

func TestElections_CleanupDisallowsNew(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()

	elector, closeFunc := Provide(iTTLStorage, coreutils.MockTime)

	ctx1 := elector.AcquireLeadership("keyA", "valA", seconds10)
	require.NotNil(t, ctx1)

	closeFunc() // cleanup => no further acquisitions

	<-ctx1.Done()
	ctx2 := elector.AcquireLeadership("keyB", "valB", seconds10)
	require.Nil(t, ctx2, "No new leadership after cleanup")
}

// TestElections_LeadershipExpires ensures that if time passes beyond the TTL and we do not renew,
// the key is automatically pruned from storage, making renewal fail.
func TestElections_LeadershipExpires(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()
	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()
	iTTLStorage.onBeforeCompareAndSwap = func() {
		coreutils.MockTime.Sleep(seconds10 * time.Second)
	}

	// Acquire leadership with a short TTL
	ctx := elector.AcquireLeadership("expireKey", "expireVal", seconds10)
	require.NotNil(t, ctx, "Should have leadership initially")

	coreutils.MockTime.Sleep(seconds10 * time.Second)

	<-ctx.Done()
}

func TestCleanupDuringRenewal(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage := newMockStorage[string, string]()
	compareAndSwapCalled := make(chan interface{})
	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	iTTLStorage.onBeforeCompareAndSwap = func() {
		close(compareAndSwapCalled)
	}

	ctx := elector.AcquireLeadership("expireKey", "expireVal", seconds10)

	{
		coreutils.MockTime.Sleep(time.Duration(seconds10) * time.Second)

		// gauarantee that <-ticker case is fired, not <-li.ctx.Done()
		// otherwise we do not know which case will fire on cleanup()
		<-compareAndSwapCalled

		// now force cancel everything.
		// successful finalizing after that shows that there are no deadlocks caused by simultaneous locks in
		// cleanup() and in releaseLeadership() after leadership loss
		cleanup()
		<-ctx.Done()
	}
}

// mockStorage is a thread-safe in-memory mock of ITTLStorage that supports key expiration.
type mockStorage[K comparable, V comparable] struct {
	mu                     sync.Mutex
	data                   map[K]valueWithTTL[V]
	expirations            map[K]time.Time // expiration time for each key
	errorTrigger           map[K]bool
	tm                     coreutils.ITime
	onBeforeCompareAndSwap func() // != nil -> called right before CompareAndSwap. Need to implement hook in tests
}

func newMockStorage[K comparable, V comparable]() *mockStorage[K, V] {
	return &mockStorage[K, V]{
		data:         make(map[K]valueWithTTL[V]),
		expirations:  make(map[K]time.Time),
		errorTrigger: make(map[K]bool),
		tm:           coreutils.MockTime,
	}
}

func (m *mockStorage[K, V]) pruneExpired() {
	now := m.tm.Now()
	for k, v := range m.data {
		if now.After(v.expiresAt) {
			delete(m.data, k)
		}
	}
}

type valueWithTTL[V any] struct {
	value     V
	expiresAt time.Time
}

// causeErrorIfNeeded simulates a forced error for specific keys
func (m *mockStorage[K, V]) causeErrorIfNeeded(key K) error {
	if m.errorTrigger[key] {
		return errors.New("forced storage error for test")
	}
	return nil
}

func (m *mockStorage[K, V]) InsertIfNotExist(key K, val V, ttlSeconds int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	// Check if key exists
	if _, exists := m.data[key]; exists {
		return false, nil
	}

	// Insert new value with TTL
	m.data[key] = valueWithTTL[V]{
		value:     val,
		expiresAt: m.tm.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	return true, nil
}

func (m *mockStorage[K, V]) CompareAndSwap(key K, oldVal V, newVal V, ttlSeconds int) (bool, error) {
	if m.onBeforeCompareAndSwap != nil {
		m.onBeforeCompareAndSwap()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	// Check if key exists and value matches
	if entry, exists := m.data[key]; !exists || entry.value != oldVal {
		return false, nil
	}

	// Update value and TTL
	m.data[key] = valueWithTTL[V]{
		value:     newVal,
		expiresAt: m.tm.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	return true, nil
}

// CompareAndDelete removes the key if current value matches val. Expiration is also removed.
func (m *mockStorage[K, V]) CompareAndDelete(key K, val V) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	entry, exists := m.data[key]
	if !exists {
		return false, nil
	}
	if entry.value != val {
		return false, nil
	}

	delete(m.data, key)
	delete(m.expirations, key)
	return true, nil
}
