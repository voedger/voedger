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
	leadership map[K]*leaderInfo[V]
}

type leaderInfo[V any] struct {
	val    V
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newElections[K comparable, V any](storage ITTLStorage[K, V], clock coreutils.ITime) *elections[K, V] {
	return &elections[K, V]{
		storage:    storage,
		clock:      clock,
		leadership: make(map[K]*leaderInfo[V]),
	}
}

// AcquireLeadership returns nil if leadership is *not* acquired (e.g., error in storage,
// already local leader, or elections cleaned up), otherwise returns a *non-nil* context.
func (e *elections[K, V]) AcquireLeadership(key K, val V, duration time.Duration) context.Context {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cleanedUp {
		logger.Verbose("[AcquireLeadership] Key=%v: elections cleaned up; cannot acquire leadership.", key)
		return nil
	}
	if _, exists := e.leadership[key]; exists {
		logger.Verbose("[AcquireLeadership] Key=%v: already leader in this instance.", key)
		return nil
	}

	// Try InsertIfNotExist
	inserted, err := e.storage.InsertIfNotExist(key, val)
	if err != nil {
		logger.Verbose("[AcquireLeadership] Key=%v: storage error: %v", key, err)
		return nil
	}
	if !inserted {
		logger.Verbose("[AcquireLeadership] Key=%v: already held or blocked by storage.", key)
		return nil
	}

	// Succeeded: create a live context
	ctx, cancel := context.WithCancel(context.Background())
	li := &leaderInfo[V]{
		val:    val,
		ctx:    ctx,
		cancel: cancel,
	}
	li.wg.Add(1)
	e.leadership[key] = li

	go e.maintainLeadership(key, val, duration, li)
	return ctx
}

func (e *elections[K, V]) maintainLeadership(key K, val V, duration time.Duration, li *leaderInfo[V]) {
	defer li.wg.Done()

	tickerInterval := duration / 2
	ticker := e.clock.NewTimerChan(tickerInterval)

	for {
		select {
		case <-li.ctx.Done():
			// Voluntarily released or forcibly canceled
			return
		case <-ticker:
			ok, err := e.storage.CompareAndSwap(key, val, val)
			if err != nil {
				logger.Verbose("[maintainLeadership] Key=%v: storage error => release", key)
				e.ReleaseLeadership(key)
				return
			}
			if !ok {
				logger.Verbose("[maintainLeadership] Key=%v: compareAndSwap failed => release", key)
				e.ReleaseLeadership(key)
				return
			}
			// Re-arm the ticker
			ticker = e.clock.NewTimerChan(tickerInterval)
		}
	}
}

func (e *elections[K, V]) ReleaseLeadership(key K) {
	e.mu.Lock()
	li, found := e.leadership[key]
	if !found {
		e.mu.Unlock()
		logger.Verbose("[ReleaseLeadership] Key=%v: not locally held.", key)
		return
	}
	delete(e.leadership, key)
	e.mu.Unlock()

	_, err := e.storage.CompareAndDelete(key, li.val)
	if err != nil {
		logger.Verbose("[ReleaseLeadership] Key=%v: storage CompareAndDelete error: %v", key, err)
	}
	li.cancel()
	li.wg.Wait()
	logger.Verbose("[ReleaseLeadership] Key=%v: leadership released.", key)
}

// cleanup disallows new acquisitions, releases all ongoing leadership, and waits for them to finish.
func (e *elections[K, V]) cleanup() {
	e.mu.Lock()
	if e.cleanedUp {
		e.mu.Unlock()
		return
	}
	e.cleanedUp = true

	keys := make([]K, 0, len(e.leadership))
	for k := range e.leadership {
		keys = append(keys, k)
	}
	e.mu.Unlock()

	// Release each leadership so renewal goroutines stop
	for _, k := range keys {
		e.ReleaseLeadership(k)
	}
}
