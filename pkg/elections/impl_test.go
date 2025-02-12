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

// mockStorage is a thread-safe in-memory mock of ITTLStorage that supports key expiration.
type mockStorage[K comparable, V comparable] struct {
	mu          sync.Mutex
	store       map[K]V
	expirations map[K]time.Time // expiration time for each key

	// We use nowFunc to get the current time (in tests, this is provided by fakeTime.Now()).
	nowFunc      func() time.Time
	errorTrigger map[K]bool
}

// newMockStorage builds a storage mock with expiration support. By default,
// nowFunc uses the real time, but in tests we'll often set it to fakeTime.Now.
func newMockStorage[K comparable, V comparable]() *mockStorage[K, V] {
	return &mockStorage[K, V]{
		store:        make(map[K]V),
		expirations:  make(map[K]time.Time),
		nowFunc:      time.Now, // can be overridden in tests
		errorTrigger: make(map[K]bool),
	}
}

func (m *mockStorage[K, V]) pruneExpired() {
	now := m.nowFunc()
	for k, exp := range m.expirations {
		if now.After(exp) {
			// Key has expired
			delete(m.store, k)
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

	if _, exists := m.store[key]; exists {
		return false, nil
	}
	m.store[key] = val
	m.expirations[key] = m.nowFunc().Add(ttl)
	return true, nil
}

// CompareAndSwap refreshes TTL if oldVal matches the current value. Otherwise fails.
func (m *mockStorage[K, V]) CompareAndSwap(key K, oldVal V, newVal V, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	curr, exists := m.store[key]
	if !exists {
		return false, nil
	}
	if curr != oldVal {
		return false, nil
	}
	m.store[key] = newVal
	m.expirations[key] = m.nowFunc().Add(ttl)
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

	curr, exists := m.store[key]
	if !exists {
		return false, nil
	}
	if curr != val {
		return false, nil
	}
	delete(m.store, key)
	delete(m.expirations, key)
	return true, nil
}

// fakeTime allows us to manually control "current time" in tests, which is essential for TTL checks.

func TestElections_AcquireLeadership_Success(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	// Make the mock storage read time from fakeTime
	storage.nowFunc = clock.Now

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("testKey", "leaderVal", 5*time.Second)
	require.NotNil(t, ctx, "Should return a non-nil context on successful acquisition")

	// Check mock storage
	storage.mu.Lock()
	val, ok := storage.store["testKey"]
	exp := storage.expirations["testKey"]
	storage.mu.Unlock()

	// wait for moment when renewal goroutine started
	<-elector.(*elections[string, string]).leadership["testKey"].renewalIsStarted
	clock.Sleep(1 * time.Second)

	require.True(t, ok, "Key should be in store")
	require.Equal(t, "leaderVal", val)
	require.True(t, clock.Now().Before(exp) || clock.Now().Equal(exp), "Expiration time should be set in the future")

	// Release leadership
	elector.ReleaseLeadership("testKey")
	<-ctx.Done()
}

func TestElections_AcquireLeadership_Failure(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	// Occupied key
	storage.mu.Lock()
	storage.store["occupied"] = "someVal"
	storage.expirations["occupied"] = clock.Now().Add(10 * time.Second)
	storage.mu.Unlock()

	ctx := elector.AcquireLeadership("occupied", "myVal", 3*time.Second)
	require.Nil(t, ctx, "Should return nil if the key is already in storage")
}

func TestElections_AcquireLeadership_Error(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	storage.errorTrigger["badKey"] = true

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("badKey", "val", 10*time.Second)
	require.Nil(t, ctx, "Should return nil if a storage error occurs")
}

func TestElections_CompareAndSwap_RenewFails(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("renewKey", "renewVal", 4*time.Second)
	require.NotNil(t, ctx)

	// wait for moment when renewal goroutine started
	<-elector.(*elections[string, string]).leadership["renewKey"].renewalIsStarted
	// trigger the renewal
	clock.Sleep(1 * time.Second)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	storage.mu.Lock()
	storage.store["renewKey"] = "someOtherVal"
	storage.mu.Unlock()

	// trigger the renewal
	clock.Sleep(5 * time.Second)

	// The leadership is forcibly released in the background.
	storage.mu.Lock()
	val, ok := storage.store["renewKey"]
	storage.mu.Unlock()

	<-ctx.Done()
	require.ErrorIs(t, ctx.Err(), context.Canceled, "Context should be canceled after renewal failure")
	require.True(t, ok, "Key should remain but with different value => we lost leadership.")
	require.Equal(t, "someOtherVal", val)
}

func TestElections_TTLStoredOnSwap(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	elector, cleanup := Provide[string, string](storage, clock)

	ctx := elector.AcquireLeadership("ttlKey", "ttlVal", 10*time.Second)
	require.NotNil(t, ctx)

	// wait for moment when renewal goroutine started
	<-elector.(*elections[string, string]).leadership["ttlKey"].renewalIsStarted
	// Trigger renewal
	clock.Sleep(5 * time.Second)

	storage.mu.Lock()
	expTime := storage.expirations["ttlKey"]
	storage.mu.Unlock()

	require.True(t, clock.Now().Before(expTime), "CompareAndSwap should have updated TTL to now+10s")
	cleanup()
	<-ctx.Done()
}

func TestElections_ReleaseLeadership_NoLeader(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	// Releasing unknown key => no effect
	elector.ReleaseLeadership("unknownKey")
}

func TestElections_CleanupDisallowsNew(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	elector, closeFunc := Provide[string, string](storage, clock)

	ctx1 := elector.AcquireLeadership("keyA", "valA", 2*time.Second)
	require.NotNil(t, ctx1)

	closeFunc() // cleanup => no further acquisitions

	require.ErrorIs(t, ctx1.Err(), context.Canceled, "Context should be canceled after cleanup")
	ctx2 := elector.AcquireLeadership("keyB", "valB", 2*time.Second)
	require.Nil(t, ctx2, "No new leadership after cleanup")
}

// TestElections_LeadershipExpires ensures that if time passes beyond the TTL and we do not renew,
// the key is automatically pruned from storage, making renewal fail.
func TestElections_LeadershipExpires(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime
	storage.nowFunc = clock.Now

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	duration := 20 * time.Millisecond
	// Acquire leadership with a short TTL
	ctx := elector.AcquireLeadership("expireKey", "expireVal", duration)
	require.NotNil(t, ctx, "Should have leadership initially")

	storage.mu.Lock()
	delete(storage.store, "expireKey") // simulate a key expiration
	storage.mu.Unlock()

	// wait for moment when renewal goroutine started
	<-elector.(*elections[string, string]).leadership["expireKey"].renewalIsStarted
	clock.Sleep(duration)

	<-ctx.Done()
}
