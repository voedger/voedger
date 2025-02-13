/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

// AcquireLeadership returns nil if leadership is *not* acquired (e.g., error in storage,
// already local leader, or elections cleaned up), otherwise returns a *non-nil* context.
func (e *elections[K, V]) AcquireLeadership(key K, val V, duration time.Duration) context.Context {
	if e.isFinalized.Load() {
		logger.Verbose("[AcquireLeadership] Key=%v: elections cleaned up; cannot acquire leadership.", key)
		return nil
	}

	inserted, err := e.storage.InsertIfNotExist(key, val, duration)
	if err != nil {
		logger.Error("[AcquireLeadership] Key=%v: storage error: %v", key, err)
		return nil
	}
	if !inserted {
		logger.Verbose("[AcquireLeadership] Key=%v: already held or blocked by storage.", key)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	li := &leaderInfo[K, V]{
		val:    val,
		ctx:    ctx,
		cancel: cancel,
	}
	e.leadership.Store(key, li)

	li.wg.Add(1)
	maintainLeadershipStarted := sync.WaitGroup{}
	maintainLeadershipStarted.Add(1)
	go e.maintainLeadership(key, val, duration, li, &maintainLeadershipStarted)
	maintainLeadershipStarted.Wait()
	return ctx
}

func (e *elections[K, V]) maintainLeadership(key K, val V, duration time.Duration, li *leaderInfo[K, V], maintainLeadershipStarted *sync.WaitGroup) {
	defer li.wg.Done()

	tickerInterval := duration / 2
	ticker := e.clock.NewTimerChan(tickerInterval)
	maintainLeadershipStarted.Done()

	for li.ctx.Err() == nil {
		select {
		case <-li.ctx.Done():
			// Voluntarily released or forcibly canceled
			return
		case <-ticker:
			logger.Verbose("[maintainLeadership] Key=%v: renewing leadership.", key)
			ok, err := e.storage.CompareAndSwap(key, val, val, duration)
			if err != nil {
				logger.Error("[maintainLeadership] Key=%v: compareAndSwap error => release", key)
			}

			if !ok {
				logger.Error("[maintainLeadership] Key=%v: compareAndSwap failed => release", key)
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
	liIntf, found := e.leadership.LoadAndDelete(key)
	if !found {
		logger.Verbose("[ReleaseLeadership] Key=%v: not locally held.", key)
		return nil
	}

	li := liIntf.(*leaderInfo[K, V])
	if _, err := e.storage.CompareAndDelete(key, li.val); err != nil {
		logger.Error("[ReleaseLeadership] Key=%v: storage CompareAndDelete error: %v", key, err)
	}

	li.cancel()

	logger.Verbose("[ReleaseLeadership] Key=%v: leadership released.", key)
	return li
}

// cleanup disallows new acquisitions, releases all ongoing leadership, and waits for them to finish.
func (e *elections[K, V]) cleanup() {
	e.isFinalized.Store(true)

	// Release each leadership so renewal goroutines stop
	e.leadership.Range(func(key, liIntf any) bool {
		li := liIntf.(*leaderInfo[K, V])
		li.cancel()
		li.wg.Wait()
		e.leadership.Delete(key)
		return true
	})
}
