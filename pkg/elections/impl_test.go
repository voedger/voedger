/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
)

// mockStorage is a simple thread-safe in-memory mock of ITTLStorage for testing.
type mockStorage[K comparable, V comparable] struct {
	mu    sync.Mutex
	store map[K]V
}

func newMockStorage[K comparable, V comparable]() *mockStorage[K, V] {
	return &mockStorage[K, V]{
		store: make(map[K]V),
	}
}

func (m *mockStorage[K, V]) InsertIfNotExist(key K, val V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.store[key]; exists {
		return false
	}
	m.store[key] = val
	return true
}

func (m *mockStorage[K, V]) CompareAndSwap(key K, oldVal V, newVal V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.store[key]
	if !exists {
		return false
	}
	// Because V is generic, equality can be done in different ways.
	// For test simplicity, assume V is comparable (e.g., string).
	if current != oldVal {
		return false
	}
	m.store[key] = newVal
	return true
}

func (m *mockStorage[K, V]) CompareAndDelete(key K, val V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.store[key]
	if !exists {
		return false
	}
	if current != val {
		return false
	}
	delete(m.store, key)
	return true
}

func TestElections_AcquireLeadership_Success(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	e, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := e.AcquireLeadership("testKey", "leaderVal", 2*time.Second)
	select {
	case <-ctx.Done():
		t.Fatal("Expected leadership context to be active, but it was canceled immediately.")
	default:
		// Context is still active, so leadership is held.
	}

	// Release leadership
	err := e.ReleaseLeadership("testKey")
	require.NoError(t, err, "ReleaseLeadership should succeed with no errors.")

	// Now ctx should be canceled
	select {
	case <-ctx.Done():
		// Good: leadership is no longer held
	default:
		t.Fatal("Expected context to be canceled after release, but it's still active.")
	}
}

func TestElections_AcquireLeadership_AfterCleanup(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	e, cleanup := Provide[string, string](storage, clock)
	cleanup() // Immediately clean up before acquiring

	ctx := e.AcquireLeadership("testKey", "val", 2*time.Second)
	select {
	case <-ctx.Done():
		// Expected: AcquireLeadership returns a canceled ctx after Cleanup.
	default:
		t.Fatal("Expected AcquireLeadership to return canceled ctx post-Cleanup.")
	}
}

func TestElections_AcquireLeadership_AlreadyLeader(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	e, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx1 := e.AcquireLeadership("duplicateKey", "val1", 2*time.Second)
	select {
	case <-ctx1.Done():
		t.Fatal("Expected first leadership to be active, but context is canceled.")
	default:
		// Good
	}

	// Attempt to acquire the same key in the same instance
	ctx2 := e.AcquireLeadership("duplicateKey", "val2", 2*time.Second)
	select {
	case <-ctx2.Done():
		require.NoError(t, ctx1.Err()) // ctx1 is still active because first leadership is still held
		// Expected, because we are already leader for "duplicateKey" in this instance.
	default:
		t.Fatal("Expected second AcquireLeadership to return canceled context for same key.")
	}

	// Clean up
	err := e.ReleaseLeadership("duplicateKey")
	require.NoError(t, err)
}

func TestElections_AcquireLeadership_InsertFails(t *testing.T) {
	// Pre-populate storage so InsertIfNotExist will fail
	storage := newMockStorage[string, string]()
	storage.store["occupiedKey"] = "existingVal"

	clock := coreutils.MockTime
	e, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	ctx := e.AcquireLeadership("occupiedKey", "val", time.Second)
	select {
	case <-ctx.Done():
		// As expected, canceled immediately
	default:
		t.Fatal("Expected AcquireLeadership to return canceled context, but it's active.")
	}
}

func TestElections_ReleaseLeadership_NoLeader(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	e, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	// Releasing when we don't hold leadership should be a no-op (no error).
	err := e.ReleaseLeadership("unknownKey")
	require.NoError(t, err, "ReleaseLeadership of an unknown key should not fail.")
}

func TestElections_Cleanup_StopsAllLeadership(t *testing.T) {
	storage := newMockStorage[string, string]()
	clock := coreutils.MockTime

	e, cleanup := Provide[string, string](storage, clock)

	ctxA := e.AcquireLeadership("keyA", "valA", 2*time.Second)
	ctxB := e.AcquireLeadership("keyB", "valB", 2*time.Second)

	// Both should be active
	select {
	case <-ctxA.Done():
		t.Fatal("KeyA leadership ended prematurely.")
	default:
	}
	select {
	case <-ctxB.Done():
		t.Fatal("KeyB leadership ended prematurely.")
	default:
	}

	cleanup()

	// After cleanup, both contexts should be canceled
	select {
	case <-ctxA.Done():
		// Good
	default:
		t.Error("Expected ctxA to be canceled after Cleanup.")
	}
	select {
	case <-ctxB.Done():
		// Good
	default:
		t.Error("Expected ctxB to be canceled after Cleanup.")
	}

	// Attempting to AcquireLeadership after cleanup => canceled context
	ctxC := e.AcquireLeadership("keyC", "valC", 2*time.Second)
	select {
	case <-ctxC.Done():
		// Good
	default:
		t.Fatal("Expected canceled context when acquiring after Cleanup.")
	}
}

func TestElections_CompareAndSwapFailure(t *testing.T) {
	// We'll let InsertIfNotExist succeed but then sabotage the stored value
	// so the next CompareAndSwap fails, simulating forced leadership loss.

	storage := newMockStorage[string, string]()
	clock := coreutils.NewITime()

	e, cleanup := Provide[string, string](storage, clock)
	defer cleanup()

	duration := 20 * time.Millisecond
	ctx := e.AcquireLeadership("keyRenew", "valRenew", duration)
	select {
	case <-ctx.Done():
		t.Fatal("Expected leadership context to be active, but it was canceled immediately.")
	default:
	}

	// force reset the timer just to cover this case
	clock.Sleep(duration)

	// Overwrite the stored value so that next CompareAndSwap("valRenew","valRenew") fails
	storage.mu.Lock()
	storage.store["keyRenew"] = "otherVal"
	storage.mu.Unlock()

	// Now simulate the periodic renewal by ticking the clock
	clock.Sleep(duration)

	// The background goroutine should detect CompareAndSwap failure and release leadership
	<-ctx.Done() // This blocks until the leadership is released

	// Check if "keyRenew" is still in storage
	storage.mu.Lock()
	defer storage.mu.Unlock()
	// We expect that CompareAndSwap failed, so ReleaseLeadership was called.
	// ReleaseLeadership does CompareAndDelete with "valRenew". That won't match the new "otherVal",
	// so the key won't get deleted from storage. The main effect is that the local leadership is lost.
	require.Contains(t, storage.store, "keyRenew", "Key is likely still there with 'otherVal'. Leadership is lost, but the key remains.")
}
