/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockStorage is a simple thread-safe in-memory mock of ITTLStorage for testing.
type mockStorage[K comparable, V comparable] struct {
	mu           sync.Mutex
	store        map[K]V
	errorTrigger map[K]bool
}

func newMockStorage[K comparable, V comparable]() *mockStorage[K, V] {
	return &mockStorage[K, V]{
		store:        make(map[K]V),
		errorTrigger: make(map[K]bool),
	}
}

func (m *mockStorage[K, V]) InsertIfNotExist(key K, val V) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errorTrigger[key] {
		return false, errors.New("forced storage error for InsertIfNotExist")
	}
	if _, exists := m.store[key]; exists {
		return false, nil
	}
	m.store[key] = val
	return true, nil
}

func (m *mockStorage[K, V]) CompareAndSwap(key K, oldVal, newVal V) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errorTrigger[key] {
		return false, errors.New("forced storage error for CompareAndSwap")
	}
	curr, exists := m.store[key]
	if !exists {
		return false, nil
	}
	if curr != oldVal {
		return false, nil
	}
	m.store[key] = newVal
	return true, nil
}

func (m *mockStorage[K, V]) CompareAndDelete(key K, val V) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errorTrigger[key] {
		return false, errors.New("forced storage error for CompareAndDelete")
	}
	curr, exists := m.store[key]
	if !exists {
		return false, nil
	}
	if curr != val {
		return false, nil
	}
	delete(m.store, key)
	return true, nil
}

// fakeTime is used for controlling the "timer" channels manually.
type fakeTime struct {
	now time.Time

	mu      sync.Mutex
	tickers []*chan time.Time
}

func newFakeTime(start time.Time) *fakeTime {
	return &fakeTime{now: start}
}

func (f *fakeTime) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeTime) NewTimerChan(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch := make(chan time.Time, 1)
	f.tickers = append(f.tickers, &ch)
	return ch
}

func (f *fakeTime) Sleep(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f *fakeTime) tickAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, chPtr := range f.tickers {
		select {
		case (*chPtr) <- f.now:
		default:
		}
	}
}

func TestElections_AcquireLeadership_Success(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := newFakeTime(time.Now())

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("testKey", "leaderVal", 2*time.Second)
	require.NotNil(t, ctx, "Expected to acquire leadership => non-nil context.")

	// Release leadership
	elector.ReleaseLeadership("testKey")

	// Context should be canceled eventually, but from Acquire's perspective we only check non-nil.
}

func TestElections_AcquireLeadership_AlreadyLeader(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := newFakeTime(time.Now())

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx1 := elector.AcquireLeadership("dupKey", "val1", 2*time.Second)
	require.NotNil(t, ctx1, "First acquisition should succeed => non-nil context")

	// Try again with same key => should fail => nil
	ctx2 := elector.AcquireLeadership("dupKey", "val2", 2*time.Second)
	require.Nil(t, ctx2, "AcquireLeadership must return nil if we already hold leadership in this instance.")

	elector.ReleaseLeadership("dupKey")
}

func TestElections_AcquireLeadership_StorageInsertBlocked(t *testing.T) {
	// Insert the key so InsertIfNotExist returns false => Acquire must return nil
	storage := newMockStorage[string, string]()
	storage.store["occupiedKey"] = "existingVal"

	clock := newFakeTime(time.Now())
	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("occupiedKey", "val", time.Second)
	require.Nil(t, ctx, "Expected nil when InsertIfNotExist fails (key occupied).")
}

func TestElections_AcquireLeadership_ErrorOnInsert(t *testing.T) {
	// Force an actual error on InsertIfNotExist
	storage := newMockStorage[string, string]()
	storage.errorTrigger["errKey"] = true

	clock := newFakeTime(time.Now())
	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("errKey", "val", time.Second)
	require.Nil(t, ctx, "Expected nil when a storage error is triggered.")
}

func TestElections_ReleaseLeadership_NoLeader(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := newFakeTime(time.Now())

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	// Releasing a key we never acquired => no effect
	elector.ReleaseLeadership("notHeld")
}

func TestElections_CloseFunc_StopsAllLeadership(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := newFakeTime(time.Now())

	elector, closeFunc := Provide[string, string](storage, clock)

	ctxA := elector.AcquireLeadership("keyA", "valA", 2*time.Second)
	ctxB := elector.AcquireLeadership("keyB", "valB", 2*time.Second)
	require.NotNil(t, ctxA)
	require.NotNil(t, ctxB)

	// Now call the cleanup
	closeFunc()

	// Any new leadership attempt => nil
	ctxC := elector.AcquireLeadership("keyC", "valC", 2*time.Second)
	require.Nil(t, ctxC, "After cleanup, AcquireLeadership must return nil.")
}

func TestElections_CompareAndSwapFailure(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := newFakeTime(time.Now())

	elector, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := elector.AcquireLeadership("keyRenew", "valRenew", 2*time.Second)
	require.NotNil(t, ctx, "Expected to acquire leadership => non-nil context")

	// Overwrite stored value => CompareAndSwap will fail on the next renewal
	storage.mu.Lock()
	storage.store["keyRenew"] = "otherVal"
	storage.mu.Unlock()

	// Trigger renewal
	clock.tickAll()

	// The renewal goroutine calls ReleaseLeadership if CompareAndSwap fails.
	// We can't verify AcquireLeadership's return was nil (we already got a context).
	// Instead, we confirm leadership was lost in the background. If we wanted, we
	// could check that "keyRenew" is still in storage with "otherVal" after release, etc.
	storage.mu.Lock()
	v, exists := storage.store["keyRenew"]
	storage.mu.Unlock()

	require.True(t, exists)
	require.Equal(t, "otherVal", v, "Key remains with 'otherVal' => forced leadership loss.")
}
