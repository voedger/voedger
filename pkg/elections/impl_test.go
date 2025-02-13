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
)

func TestElections_BasicUsage(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("testKey", "leaderVal", 5*time.Second)
	require.NotNil(t, ctx, "Should return a non-nil context on successful acquisition")

	// Check mock storage
	storage.mu.Lock()
	val, ok := storage.data["testKey"]
	exp := storage.expirations["testKey"]
	storage.mu.Unlock()

	require.True(t, ok, "Key should be in store")
	require.Equal(t, "leaderVal", val)
	require.True(t, clock.Now().Before(exp), "Expiration time should be set in the future")

	// Release leadership
	elector.ReleaseLeadership("testKey")
	<-ctx.Done()
}

func TestElections_AcquireLeadership_Failure(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	// Occupied key
	storage.mu.Lock()
	storage.data["occupied"] = "someVal"
	storage.expirations["occupied"] = clock.Now().Add(10 * time.Second)
	storage.mu.Unlock()

	ctx := elector.AcquireLeadership("occupied", "myVal", 3*time.Second)
	require.Nil(t, ctx, "Should return nil if the key is already in storage")
}

func TestElections_AcquireLeadership_Error(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	storage.errorTrigger["badKey"] = true

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("badKey", "val", 10*time.Second)
	require.Nil(t, ctx, "Should return nil if a storage error occurs")
}

func TestElections_CompareAndSwap_RenewFails(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("renewKey", "renewVal", 4*time.Second)
	require.NotNil(t, ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	storage.mu.Lock()
	storage.data["renewKey"] = "someOtherVal"
	storage.mu.Unlock()

	// trigger the renewal
	clock.Sleep(5 * time.Second)

	// The leadership is forcibly released in the background.
	storage.mu.Lock()
	val, ok := storage.data["renewKey"]
	storage.mu.Unlock()

	<-ctx.Done()
	require.ErrorIs(t, ctx.Err(), context.Canceled, "Context should be canceled after renewal failure")
	require.True(t, ok, "Key should remain but with different value => we lost leadership.")
	require.Equal(t, "someOtherVal", val)
}

func TestElections_ReleaseLeadership_NoLeader(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	// Releasing unknown key => no effect
	elector.ReleaseLeadership("unknownKey")
}

func TestElections_CleanupDisallowsNew(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	elector, closeFunc := Provide[string, string](storage, clock)

	ctx1 := elector.AcquireLeadership("keyA", "valA", 2*time.Second)
	require.NotNil(t, ctx1)

	closeFunc() // cleanup => no further acquisitions

	<-ctx1.Done()
	ctx2 := elector.AcquireLeadership("keyB", "valB", 2*time.Second)
	require.Nil(t, ctx2, "No new leadership after cleanup")
}

// TestElections_LeadershipExpires ensures that if time passes beyond the TTL and we do not renew,
// the key is automatically pruned from storage, making renewal fail.
func TestElections_LeadershipExpires(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	duration := 20 * time.Millisecond
	// Acquire leadership with a short TTL
	ctx := elector.AcquireLeadership("expireKey", "expireVal", duration)
	require.NotNil(t, ctx, "Should have leadership initially")

	storage.mu.Lock()
	delete(storage.data, "expireKey") // simulate a key expiration
	storage.mu.Unlock()

	clock.Sleep(duration)

	<-ctx.Done()
}

func TestCleanupDuringRenewal(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	compareAndSwapCalled := make(chan interface{})
	storage.onCompareAndSwap = func() {
		close(compareAndSwapCalled)
	}

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	duration := 20 * time.Millisecond
	ctx := elector.AcquireLeadership("expireKey", "expireVal", duration)

	{
		storage.mu.Lock()
		delete(storage.data, "expireKey") // simulate a key expiration
		storage.mu.Unlock()

		clock.Sleep(duration)

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
	mu               sync.Mutex
	data             map[K]V
	expirations      map[K]time.Time // expiration time for each key
	errorTrigger     map[K]bool
	tm               coreutils.ITime
	onCompareAndSwap func() // != nil -> called right before CompareAndSwap. Need to implement hook in tests
}

// newMockStorage builds a storage mock with expiration support. By default,
// tm.Now uses the real time, but in tests we'll often set it to fakeTime.Now.
func newMockStorage[K comparable, V comparable]() *mockStorage[K, V] {
	return &mockStorage[K, V]{
		data:         make(map[K]V),
		expirations:  make(map[K]time.Time),
		errorTrigger: make(map[K]bool),
		tm:           coreutils.MockTime,
	}
}

func (m *mockStorage[K, V]) pruneExpired() {
	now := m.tm.Now()
	for k, exp := range m.expirations {
		if now.After(exp) {
			// Key has expired
			delete(m.data, k)
			delete(m.expirations, k)
		}
	}
}

// causeErrorIfNeeded simulates a forced error for specific keys
func (m *mockStorage[K, V]) causeErrorIfNeeded(key K) error {
	if m.errorTrigger[key] {
		return errors.New("forced storage error for test")
	}
	return nil
}

// InsertIfNotExist checks for expiration, then inserts key if absent, setting its TTL.
func (m *mockStorage[K, V]) InsertIfNotExist(key K, val V, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = val
	m.expirations[key] = m.tm.Now().Add(ttl)
	return true, nil
}

// CompareAndSwap refreshes TTL if oldVal matches the current value. Otherwise fails.
func (m *mockStorage[K, V]) CompareAndSwap(key K, oldVal V, newVal V, ttl time.Duration) (bool, error) {
	if m.onCompareAndSwap != nil {
		m.onCompareAndSwap()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	curr, exists := m.data[key]
	if !exists {
		return false, nil
	}
	if curr != oldVal {
		return false, nil
	}
	m.data[key] = newVal
	m.expirations[key] = m.tm.Now().Add(ttl)
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

	curr, exists := m.data[key]
	if !exists {
		return false, nil
	}
	if curr != val {
		return false, nil
	}
	delete(m.data, key)
	delete(m.expirations, key)
	return true, nil
}
