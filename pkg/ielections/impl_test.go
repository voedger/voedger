/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

// [~server.design.orch/ElectionsTest~impl]
func TestElections(t *testing.T) {
	ttlStorage := newTTLStorageMock[string, string]()
	counter := 0
	ElectionsTestSuite(t, ttlStorage, TestDataGen[string, string]{
		NextKey: func() string {
			counter++
			return "testKey" + strconv.Itoa(counter)
		},
		NextVal: func() string {
			counter++
			return "testVal" + strconv.Itoa(counter)
		},
	})
}

func TestTransientErrorRecovery(t *testing.T) {
	require := require.New(t)
	storage := newTTLStorageMock[string, string]()
	elections, cleanup := Provide(storage, testingu.MockTime)
	defer cleanup()

	const leadershipDuration LeadershipDurationSeconds = 40
	ctx := elections.AcquireLeadership("k", "v", leadershipDuration)
	require.NotNil(ctx)

	const totalErrorRetries = 3
	retryTimerArmed := make(chan struct{}, 1)
	casSucceeded := make(chan struct{}, 1)
	var casCallCount atomic.Int32
	storage.onBeforeCompareAndSwap = func() {
		cnt := int(casCallCount.Add(1))
		if cnt == totalErrorRetries+1 {
			storage.mu.Lock()
			delete(storage.errorTrigger, "k")
			storage.mu.Unlock()
		}
		if cnt <= totalErrorRetries {
			testingu.MockTime.SetOnNextTimerArmed(func() {
				retryTimerArmed <- struct{}{}
			})
		} else {
			casSucceeded <- struct{}{}
		}
	}

	storage.mu.Lock()
	storage.errorTrigger["k"] = true
	storage.mu.Unlock()

	tickerInterval := time.Duration(leadershipDuration) * time.Second / renewalsPerLeadershipDur
	testingu.MockTime.Sleep(tickerInterval)

	for range totalErrorRetries {
		<-retryTimerArmed
		require.NoError(ctx.Err(), "leadership must be retained during retries")
		testingu.MockTime.Sleep(time.Second)
	}

	<-casSucceeded

	require.Equal(int32(totalErrorRetries+1), casCallCount.Load())
	require.NoError(ctx.Err(), "leadership must be retained after transient error recovery")

	elections.ReleaseLeadership("k")
	<-ctx.Done()
}

func TestAcuireLeadershipFailureAfterCompareAndSwapError(t *testing.T) {
	require := require.New(t)
	storage := newTTLStorageMock[string, string]()
	elections, cleanup := Provide(storage, testingu.MockTime)
	defer cleanup()

	const leadershipDuration LeadershipDurationSeconds = 40
	ctx := elections.AcquireLeadership("k", "v", leadershipDuration)
	require.NotNil(ctx)

	retryTimerArmed := make(chan struct{}, 1)
	var casCallCount atomic.Int32
	storage.onBeforeCompareAndSwap = func() {
		casCallCount.Add(1)
		testingu.MockTime.SetOnNextTimerArmed(func() {
			retryTimerArmed <- struct{}{}
		})
	}

	storage.mu.Lock()
	storage.errorTrigger["k"] = true
	storage.mu.Unlock()

	tickerInterval := time.Duration(leadershipDuration) * time.Second / renewalsPerLeadershipDur
	testingu.MockTime.Sleep(tickerInterval)

	const expectedMinRetries = 5
	for range expectedMinRetries {
		<-retryTimerArmed
		require.NoError(ctx.Err(), "leadership must be retained during retries")
		testingu.MockTime.Sleep(time.Second)
	}

	require.GreaterOrEqual(casCallCount.Load(), int32(expectedMinRetries))

	testingu.MockTime.Sleep(tickerInterval)

	<-ctx.Done()
	require.Error(ctx.Err(), "leadership must be lost after persistent errors")
}

func TestCleanupDuringCompareAndSwapRetries(t *testing.T) {
	require := require.New(t)
	storage := newTTLStorageMock[string, string]()
	elections, cleanup := Provide(storage, testingu.MockTime)

	const leadershipDuration LeadershipDurationSeconds = 40
	ctx := elections.AcquireLeadership("k", "v", leadershipDuration)
	require.NotNil(ctx)

	retryTimerArmed := make(chan struct{}, 1)
	var casCallCount atomic.Int32
	storage.onBeforeCompareAndSwap = func() {
		casCallCount.Add(1)
		testingu.MockTime.SetOnNextTimerArmed(func() {
			retryTimerArmed <- struct{}{}
		})
	}

	storage.mu.Lock()
	storage.errorTrigger["k"] = true
	storage.mu.Unlock()

	tickerInterval := time.Duration(leadershipDuration) * time.Second / renewalsPerLeadershipDur
	testingu.MockTime.Sleep(tickerInterval)

	const expectedMinRetries = 3
	for range expectedMinRetries {
		<-retryTimerArmed
		require.NoError(ctx.Err(), "leadership must be retained during retries")
		testingu.MockTime.Sleep(time.Second)
	}

	<-retryTimerArmed
	require.GreaterOrEqual(casCallCount.Load(), int32(expectedMinRetries))

	cleanup()

	<-ctx.Done()
	require.Error(ctx.Err())
}
