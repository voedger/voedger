/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

func Test_newKillerScheduler(t *testing.T) {
	require := require.New(t)
	mockTime := testingu.NewMockTime()
	ks := newKillerScheduler(mockTime)
	require.NotNil(ks)
	require.NotNil(ks.clock)

	// Verify that the clock is isolated by checking it's a different instance
	// Advance global MockTime and verify isolated clock is not affected
	initialTime := ks.clock.Now()
	mockTime.Add(10 * time.Second)
	require.Equal(initialTime, ks.clock.Now())
}

func Test_scheduleKiller(t *testing.T) {

	t.Run("does not schedule killer for wrong deadline", func(t *testing.T) {
		require := require.New(t)
		mockTime := testingu.NewMockTime()
		ks := newKillerScheduler(mockTime)

		t.Run("past deadline", func(t *testing.T) {

			deadline := ks.clock.Now().Add(-10 * time.Second)
			ks.scheduleKiller(deadline)

			require.Nil(ks.ctx)
			require.Nil(ks.cancel)
		})

		t.Run("zero duration", func(t *testing.T) {
			deadline := ks.clock.Now()
			ks.scheduleKiller(deadline)

			require.Nil(ks.ctx)
			require.Nil(ks.cancel)
		})
	})

	t.Run("cancels previous killer when scheduling new one", func(t *testing.T) {
		require := require.New(t)
		mockTime := testingu.NewMockTime()
		ks := newKillerScheduler(mockTime)

		// Schedule first killer
		deadline1 := ks.clock.Now().Add(10 * time.Second)
		ks.scheduleKiller(deadline1)
		firstCtx := ks.ctx
		require.NotNil(firstCtx)
		require.NoError(firstCtx.Err())

		// Schedule second killer
		deadline2 := ks.clock.Now().Add(20 * time.Second)
		ks.scheduleKiller(deadline2)
		secondCtx := ks.ctx
		require.NotNil(secondCtx)

		// First context should be cancelled
		require.Error(firstCtx.Err())
		// Second context should be active
		require.NoError(secondCtx.Err())
	})

}

func Test_killerRescheduleOnCAS(t *testing.T) {
	const leadershipDuration LeadershipDurationSeconds = seconds10

	newTestElections := func(storage *ttlStorageMock[string, string]) (*elections[string, string], *killerScheduler, *leaderInfo[string, string], *sync.WaitGroup) {
		e := &elections[string, string]{
			storage: storage,
			clock:   testingu.MockTime,
		}
		killer := newKillerScheduler(testingu.MockTime)
		killer.scheduleKiller(killer.clock.Now().Add(durationMult(leadershipDuration, killDeadlineFactor)))

		ctx, cancel := context.WithCancel(context.Background())
		li := &leaderInfo[string, string]{val: "v1", ctx: ctx, cancel: cancel}
		started := &sync.WaitGroup{}
		started.Add(1)
		li.wg.Add(1)
		return e, killer, li, started
	}

	t.Run("CAS success reschedules killer", func(t *testing.T) {
		require := require.New(t)
		storage := newTTLStorageMock[string, string]()
		storage.data["k1"] = valueWithTTL[string]{value: "v1", expiresAt: testingu.MockTime.Now().Add(time.Duration(leadershipDuration) * time.Second)}
		e, killer, li, started := newTestElections(storage)
		e.leadership.Store("k1", li)

		var preCASCtx context.Context
		casEntered := make(chan struct{})
		storage.onBeforeCompareAndSwap = func() {
			preCASCtx = killer.ctx
			close(casEntered)
		}

		go e.maintainLeadership("k1", "v1", leadershipDuration, li, started, killer)
		started.Wait()

		testingu.MockTime.Sleep(time.Duration(leadershipDuration) * time.Second / maintainIntervalDivisor)
		<-casEntered

		li.cancel()
		li.wg.Wait()

		require.NotNil(preCASCtx)
		require.Error(preCASCtx.Err(), "pre-CAS killer ctx must be cancelled after successful CAS reschedules killer")
		require.NotNil(killer.ctx)
		require.NoError(killer.ctx.Err(), "post-CAS killer ctx must be active")
	})

	t.Run("CAS not ok does not reschedule killer", func(t *testing.T) {
		require := require.New(t)
		storage := newTTLStorageMock[string, string]()
		storage.data["k1"] = valueWithTTL[string]{value: "v1", expiresAt: testingu.MockTime.Now().Add(time.Duration(leadershipDuration) * time.Second)}
		e, killer, li, started := newTestElections(storage)
		e.leadership.Store("k1", li)

		var preCASCtx context.Context
		storage.onBeforeCompareAndSwap = func() {
			preCASCtx = killer.ctx
			storage.mu.Lock()
			storage.data["k1"] = valueWithTTL[string]{value: "sabotaged", expiresAt: testingu.MockTime.Now().Add(20 * time.Second)}
			storage.mu.Unlock()
		}

		go e.maintainLeadership("k1", "v1", leadershipDuration, li, started, killer)
		started.Wait()

		testingu.MockTime.Sleep(time.Duration(leadershipDuration) * time.Second / maintainIntervalDivisor)

		li.wg.Wait()

		require.NotNil(preCASCtx)
		require.NoError(preCASCtx.Err(), "pre-CAS killer ctx must stay active when CAS returns !ok")
		require.Equal(preCASCtx, killer.ctx, "killer ctx must not change when CAS returns !ok")
	})

	t.Run("CAS error after all retries does not reschedule killer", func(t *testing.T) {
		require := require.New(t)
		storage := newTTLStorageMock[string, string]()
		storage.data["k1"] = valueWithTTL[string]{value: "v1", expiresAt: testingu.MockTime.Now().Add(time.Duration(leadershipDuration) * time.Second)}
		e, killer, li, started := newTestElections(storage)
		e.leadership.Store("k1", li)

		var preCASCtx context.Context
		storage.onBeforeCompareAndSwap = func() {
			if preCASCtx == nil {
				preCASCtx = killer.ctx
			}
			storage.errorTrigger["k1"] = true
		}

		go e.maintainLeadership("k1", "v1", leadershipDuration, li, started, killer)
		started.Wait()

		testingu.MockTime.Sleep(time.Duration(leadershipDuration) * time.Second / maintainIntervalDivisor)

		li.wg.Wait()

		require.NotNil(preCASCtx)
		require.NoError(preCASCtx.Err(), "pre-CAS killer ctx must stay active when CAS errors exhaust retries")
		require.Equal(preCASCtx, killer.ctx, "killer ctx must not change when CAS errors exhaust retries")
	})
}
