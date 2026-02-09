/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

// [~server.design.orch/elections~impl]

// AcquireLeadership returns nil if leadership is *not* acquired (e.g., error in storage,
// already local leader, or elections cleaned up), otherwise returns a *non-nil* context.
func (e *elections[K, V]) AcquireLeadership(key K, val V, leadershipDurationSeconds LeadershipDurationSeconds) context.Context {
	if e.isFinalized.Load() {
		logger.Verbose(fmt.Sprintf("Key=%v: elections cleaned up; cannot acquire leadership", key))
		return nil
	}

	leadershipStartTime := e.clock.Now()
	inserted, err := e.storage.InsertIfNotExist(key, val, int(leadershipDurationSeconds))
	if err != nil {
		// notest
		logger.Error(fmt.Sprintf("Key=%v: InsertIfNotExist failed: %v", key, err))
		return nil
	}
	if !inserted {
		logger.Verbose(fmt.Sprintf("Key=%v: leadership already acquired by someone else", key))
		return nil
	}

	logger.Info(fmt.Sprintf("Key=%v: leadership acquired", key))

	killer := newKillerScheduler(e.clock)
	killTime := leadershipStartTime.Add(durationMult(leadershipDurationSeconds, killDeadlineFactor))
	killer.scheduleKiller(killTime)

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
	go e.maintainLeadership(key, val, leadershipDurationSeconds, li, &maintainLeadershipStarted, killer)
	maintainLeadershipStarted.Wait()
	return ctx
}

func (e *elections[K, V]) maintainLeadership(key K, val V, leadershipDurationSeconds LeadershipDurationSeconds, li *leaderInfo[K, V], maintainLeadershipStarted *sync.WaitGroup, killer *killerScheduler) {
	defer li.wg.Done()

	maintainLeadershipInterval := time.Duration(leadershipDurationSeconds) * time.Second / maintainIntervalDivisor
	retryOnCASErrInterval := time.Duration(leadershipDurationSeconds) * time.Second / retryIntervalDivisor
	maintainLeadershipTimer := e.clock.NewTimerChan(maintainLeadershipInterval)
	maintainLeadershipStarted.Done()
	tickerCounter := int64(0)

	for li.ctx.Err() == nil {
		select {
		case <-li.ctx.Done():
			return
		case <-maintainLeadershipTimer:
			maintainLeadershipTimer = e.clock.NewTimerChan(maintainLeadershipInterval)
			tickerCounter = bumpTickerCounter(tickerCounter, key, maintainLeadershipInterval)

			leadershipStartTime := e.clock.Now()

			// Before compareAndSwap run killer with 0.75 * LeadershipDurationSeconds
			killer.scheduleKiller(leadershipStartTime.Add(durationMult(leadershipDurationSeconds, preCASKillTimeFactor)))

			var ok bool
			var err error
			for attempt := range maxRetriesOnCASErr + 1 {
				ok, err = e.storage.CompareAndSwap(key, val, val, int(leadershipDurationSeconds))
				if err == nil {
					break
				}
				logger.Error(fmt.Sprintf("Key=%v: compareAndSwap attempt %d failed: %v", key, attempt+1, err))
				if attempt < maxRetriesOnCASErr {
					e.clock.Sleep(retryOnCASErrInterval)
				}
			}

			if err != nil {
				logger.Error(fmt.Sprintf("Key=%v: compareAndSwap failed after %d attempts => release",
					key, maxRetriesOnCASErr+1))
				e.releaseLeadership(key)
				return
			}

			if ok {
				killTime := leadershipStartTime.Add(durationMult(leadershipDurationSeconds, killDeadlineFactor))
				killer.scheduleKiller(killTime)
			} else {
				logger.Error(fmt.Sprintf("Key=%v: compareAndSwap !ok => release", key))
				e.releaseLeadership(key)
				return
			}
		}
	}
}

func durationMult(seconds LeadershipDurationSeconds, factor float64) time.Duration {
	return time.Duration(float64(time.Duration(seconds)*time.Second) * factor)
}

// nolint: revive
func bumpTickerCounter[K any](tickerCounter int64, key K, tickerInterval time.Duration) (tickerCounterPlus1 int64) {
	tickerCounterPlus1 = tickerCounter + 1
	if tickerCounter < 10 {
		logger.Verbose(fmt.Sprintf("Key=%v: renewing leadership", key))
	} else if tickerCounter%200 == 0 {
		// notest
		logger.Verbose(fmt.Sprintf("Key=%v: still leader for %s", key, tickerInterval*time.Duration(tickerCounter)))
	}
	return tickerCounterPlus1
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
		logger.Verbose(fmt.Sprintf("Key=%v: we're not the leader", key))
		return nil
	}

	li := liIntf.(*leaderInfo[K, V])
	if _, err := e.storage.CompareAndDelete(key, li.val); err != nil {
		// notest
		logger.Error(fmt.Sprintf("Key=%v: CompareAndDelete failed: %v", key, err))
	}

	li.cancel()

	logger.Verbose(fmt.Sprintf("Key=%v: leadership released", key))
	return li
}

// cleanup disallows new acquisitions, releases all ongoing leadership, and waits for them to finish.
func (e *elections[K, V]) cleanup() {
	e.isFinalized.Store(true)

	// Release each leadership so renewal goroutines stop
	e.leadership.Range(func(key, liIntf any) bool {
		li := liIntf.(*leaderInfo[K, V])
		if _, err := e.storage.CompareAndDelete(key.(K), li.val); err != nil {
			// notest
			logger.Error(fmt.Sprintf("Key=%v: CompareAndDelete failed: %v", key, err))
		}
		li.cancel()
		li.wg.Wait()
		e.leadership.Delete(key)
		return true
	})
}
