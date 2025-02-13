/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

// AcquireLeadership returns nil if leadership is *not* acquired (e.g., error in storage,
// already local leader, or elections cleaned up), otherwise returns a *non-nil* context.
func (e *elections[K, V]) AcquireLeadership(key K, val V, duration time.Duration) context.Context {
	e.mu.Lock()
	if e.isFinalized {
		logger.Verbose("[AcquireLeadership] Key=%v: elections cleaned up; cannot acquire leadership.", key)
		e.mu.Unlock()
		return nil
	}
	e.mu.Unlock()

	inserted, err := e.storage.InsertIfNotExist(key, val, duration)
	if err != nil {
		logger.Verbose("[AcquireLeadership] Key=%v: storage error: %v", key, err)
		return nil
	}
	if !inserted {
		logger.Verbose("[AcquireLeadership] Key=%v: already held or blocked by storage.", key)
		return nil
	}

	e.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	li := &leaderInfo[K, V]{
		val:    val,
		ctx:    ctx,
		cancel: cancel,
	}
	e.leadership[key] = li
	e.mu.Unlock()

	li.wg.Add(1) // For the renewal goroutine
	go e.maintainLeadership(key, val, duration, li)
	// Wait for the renewal goroutine to start
	<-li.renewalIsStarted
	return ctx
}

func (e *elections[K, V]) maintainLeadership(key K, val V, duration time.Duration, li *leaderInfo[K, V]) {
	defer li.wg.Done()

	tickerInterval := duration / 2
	ticker := e.clock.NewTimerChan(tickerInterval)

	for li.ctx.Err() == nil {
		select {
		case <-li.ctx.Done():
			// Voluntarily released or forcibly canceled
			return
		case <-ticker:
			logger.Verbose("[maintainLeadership] Key=%v: renewing leadership.", key)
			ok, err := e.storage.CompareAndSwap(key, val, val, duration)
			if err != nil {
				logger.Verbose("[maintainLeadership] Key=%v: storage error => release", key)
			}

			if !ok {
				logger.Verbose("[maintainLeadership] Key=%v: compareAndSwap failed => release", key)
			}

			if !ok || err != nil {
				e.releaseLeadership(key)
				return
			}
		}
	}
}

func (e *elections[K, V]) ReleaseLeadership(key K) {
	li := e.releaseLeadership(key)
	if li == nil {
		return
	}

	li.wg.Wait()
}

func (e *elections[K, V]) releaseLeadership(key K) *leaderInfo[K, V] {
	e.mu.Lock()
	li, found := e.leadership[key]
	if !found {
		e.mu.Unlock()
		logger.Verbose("[ReleaseLeadership] Key=%v: not locally held.", key)
		return nil
	}
	li.cancel()
	delete(e.leadership, key)
	e.mu.Unlock()

	if _, err := e.storage.CompareAndDelete(key, li.val); err != nil {
		logger.Verbose("[ReleaseLeadership] Key=%v: storage CompareAndDelete error: %v", key, err)
	}

	logger.Verbose("[ReleaseLeadership] Key=%v: leadership released.", key)
	return li
}

// cleanup disallows new acquisitions, releases all ongoing leadership, and waits for them to finish.
func (e *elections[K, V]) cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.isFinalized = true
	// Release each leadership so renewal goroutines stop
	for key, li := range e.leadership {
		li.cancel()
		li.wg.Wait()
		delete(e.leadership, key)
	}
}
