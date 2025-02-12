/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

type elections[K comparable, V any] struct {
	storage ITTLStorage[K, V]
	clock   coreutils.ITime

	mu         sync.Mutex
	cleanedUp  bool
	leadership map[K]*leaderInfo[K, V]
}

// leaderInfo holds per-key tracking data for a leadership.
type leaderInfo[K comparable, V any] struct {
	val    V
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // used to wait for the renewal goroutine
}

func newElections[K comparable, V any](storage ITTLStorage[K, V], clock coreutils.ITime) *elections[K, V] {
	return &elections[K, V]{
		storage:    storage,
		clock:      clock,
		leadership: make(map[K]*leaderInfo[K, V]),
	}
}

// AcquireLeadership attempts to become the leader for `key` using `val`. It returns
// a context that remains valid while leadership is held. If leadership is not
// acquired (due to cleanup, already holding, or insert failing), it returns a
// canceled context immediately and logs the corresponding error message.
func (e *elections[K, V]) AcquireLeadership(key K, val V, duration time.Duration) context.Context {
	// Always create a new context for this attempt
	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	defer e.mu.Unlock()

	// If elections were cleaned up, we cannot acquire new leadership.
	if e.cleanedUp {
		logger.Verbose("[AcquireLeadership] Failed for key=%v: elections already cleaned up.", key)
		cancel()
		return ctx
	}

	// If we already hold leadership for this key in our local instance, do not re-acquire.
	if _, exists := e.leadership[key]; exists {
		logger.Verbose("[AcquireLeadership] Failed for key=%v: already leader in current instance.", key)
		cancel()
		return ctx
	}

	// Attempt to insert (key,val) in storage. If insertion fails, someone else holds leadership.
	inserted := e.storage.InsertIfNotExist(key, val)
	if !inserted {
		logger.Verbose("[AcquireLeadership] Failed for key=%v: storage insert blocked (held by another?).", key)
		cancel()
		return ctx
	}

	// Successfully inserted the key; we are now the leader until proven otherwise.
	li := &leaderInfo[K, V]{
		val:    val,
		ctx:    ctx,
		cancel: cancel,
	}
	// We'll run 1 background goroutine for periodic renewal
	li.wg.Add(1)

	e.leadership[key] = li

	// Start background renewal
	go e.maintainLeadership(key, val, duration, li)

	// Return the context that remains active while leadership is held
	return ctx
}

// maintainLeadership periodically calls CompareAndSwap to confirm
// that we still hold leadership. If that fails, we release leadership.
func (e *elections[K, V]) maintainLeadership(key K, val V, duration time.Duration, li *leaderInfo[K, V]) {
	defer li.wg.Done()

	renewInterval := duration / 2
	ticker := e.clock.NewTimerChan(renewInterval)

	for {
		select {
		case <-li.ctx.Done():
			// If context is canceled, leadership was voluntarily released
			return
		case <-ticker:
			// Attempt to renew leadership by CompareAndSwap(key, oldVal, newVal).
			ok := e.storage.CompareAndSwap(key, val, val)
			if !ok {
				logger.Verbose("[maintainLeadership] Leadership lost for key=%v. Releasing...", key)
				_ = e.ReleaseLeadership(key)
				return
			}
			// Refresh the ticker
			ticker = e.clock.NewTimerChan(renewInterval)
		}
	}
}

// ReleaseLeadership releases the leader position for `key`, stops the
// renewal goroutine, and attempts to CompareAndDelete the key in storage.
func (e *elections[K, V]) ReleaseLeadership(key K) error {
	e.mu.Lock()
	li, ok := e.leadership[key]
	if !ok {
		e.mu.Unlock()
		logger.Verbose("[ReleaseLeadership] No leadership found for key=%v", key)
		return nil // Not an error: we simply have no leadership to release
	}
	delete(e.leadership, key)
	e.mu.Unlock()

	// Remove from storage if still matching the original val
	e.storage.CompareAndDelete(key, li.val)

	// Signal the background goroutine to stop and wait for it
	li.cancel()
	li.wg.Wait()
	logger.Verbose("[ReleaseLeadership] Released leadership for key=%v", key)
	return nil
}

// cleanup marks the election instance as done, stops all renewal goroutines,
// and waits for them to terminate.
func (e *elections[K, V]) cleanup() {
	e.mu.Lock()
	if e.cleanedUp {
		e.mu.Unlock()
		return
	}
	e.cleanedUp = true

	toRelease := make([]K, 0, len(e.leadership))
	for key := range e.leadership {
		toRelease = append(toRelease, key)
	}
	e.mu.Unlock()

	// Release each leadership in turn. This cancels goroutines, clears storage.
	for _, key := range toRelease {
		_ = e.ReleaseLeadership(key)
	}
}
