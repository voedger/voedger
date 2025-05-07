/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

const seconds10 = 10

// [~server.design.orch/ElectionsTestSuite~impl]
func ElectionsTestSuite[K any, V any](t *testing.T, ttlStorage ITTLStorage[K, V], testData TestDataGen[K, V]) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()
	tests := map[string]func(*require.Assertions, IElections[K, V], ITTLStorage[K, V], func(), TestDataGen[K, V]){
		"BasicUsage":                                     basicUsage[K, V],
		"AcquireLeadershipIfAcquiredAlready":             acquireLeadershipIfAcquiredAlready[K, V],
		"CloseContextOnCompareAndSwapFailure_KeyChanged": closeContextOnCompareAndSwapFailure_KeyChanged[K, V],
		"CloseContextOnCompareAndSwapFailure_KeyDeleted": closeContextOnCompareAndSwapFailure_KeyDeleted[K, V],
		"ReleaseLeadershipWithoutAcquire":                releaseLeadershipWithoutAcquire[K, V],
		"AcquireFailingAfterCleanup":                     acquireFailingAfterCleanup[K, V],
		"CleanupDuringRenewal":                           cleanupDuringRenewal[K, V],
	}
	// note: testing the case when ttl record is expired is nonsense because it is renewing, can not be expired. Key deletion case is covered
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			elections, cleanup := Provide(ttlStorage, testingu.MockTime)
			defer cleanup()
			require := require.New(t)
			test(require, elections, ttlStorage, cleanup, testData)
		})
	}
}

type TestDataGen[K any, V any] struct {
	NextKey func() K
	NextVal func() V
}

func basicUsage[K any, V any](require *require.Assertions, elections IElections[K, V], iTTLStorage ITTLStorage[K, V], _ func(), dataGen TestDataGen[K, V]) {
	key := dataGen.NextKey()
	val := dataGen.NextVal()
	ctx := elections.AcquireLeadership(key, val, seconds10)
	require.NotNil(ctx, "Should return a non-nil context on successful acquisition")

	ok, storedData, err := iTTLStorage.Get(key)
	require.NoError(err)
	require.True(ok)
	require.Equal(val, storedData)

	// Release leadership
	elections.ReleaseLeadership(key)
	<-ctx.Done()
}

func acquireLeadershipIfAcquiredAlready[K any, V any](require *require.Assertions, elections IElections[K, V], iTTLStorage ITTLStorage[K, V], _ func(), dataGen TestDataGen[K, V]) {
	key := dataGen.NextKey()
	val1 := dataGen.NextVal()
	val2 := dataGen.NextVal()
	ok, err := iTTLStorage.InsertIfNotExist(key, val1, seconds10)
	require.NoError(err)
	require.True(ok)
	ctx := elections.AcquireLeadership(key, val2, seconds10)
	require.Nil(ctx, "Should return nil if the key is already in storage")
}

func closeContextOnCompareAndSwapFailure_KeyChanged[K any, V any](require *require.Assertions, elections IElections[K, V], iTTLStorage ITTLStorage[K, V], _ func(), dataGen TestDataGen[K, V]) {
	key := dataGen.NextKey()
	val1 := dataGen.NextVal()
	val2 := dataGen.NextVal()
	ctx := elections.AcquireLeadership(key, val1, seconds10)
	require.NotNil(ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	ok, err := iTTLStorage.CompareAndSwap(key, val1, val2, seconds10*2)
	require.NoError(err)
	require.True(ok)

	// trigger the renewal
	testingu.MockTime.Sleep((seconds10) * time.Second)

	// The leadership is forcibly released in the background.
	<-ctx.Done()

	// expect that the new value is still stored in the storage
	ok, keptValue, err := iTTLStorage.Get(key)
	require.NoError(err)
	require.True(ok)
	require.Equal(val2, keptValue)

	// make the sabotaged key be expired
	testingu.MockTime.Sleep((seconds10 + 1) * time.Second)
}

func closeContextOnCompareAndSwapFailure_KeyDeleted[K any, V any](require *require.Assertions, elections IElections[K, V], iTTLStorage ITTLStorage[K, V], _ func(), dataGen TestDataGen[K, V]) {
	key := dataGen.NextKey()
	val := dataGen.NextVal()
	ctx := elections.AcquireLeadership(key, val, seconds10)
	require.NotNil(ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	ok, err := iTTLStorage.CompareAndDelete(key, val)
	require.NoError(err)
	require.True(ok)

	// trigger the renewal
	testingu.MockTime.Sleep((seconds10 + 1) * time.Second)

	// The leadership is forcibly released in the background.
	<-ctx.Done()

	// just check the key is deleted indeed
	ok, _, err = iTTLStorage.Get(key)
	require.NoError(err)
	require.False(ok)
}

func releaseLeadershipWithoutAcquire[K any, V any](require *require.Assertions, elections IElections[K, V], iTTLStorage ITTLStorage[K, V], _ func(), dataGen TestDataGen[K, V]) {
	// Releasing unknown key => nothing happens, no errors
	elections.ReleaseLeadership(dataGen.NextKey())
}

func acquireFailingAfterCleanup[K any, V any](require *require.Assertions, elections IElections[K, V], iTTLStorage ITTLStorage[K, V], cleanup func(), dataGen TestDataGen[K, V]) {
	key1 := dataGen.NextKey()
	val1 := dataGen.NextVal()
	key2 := dataGen.NextKey()
	val2 := dataGen.NextVal()
	ctx1 := elections.AcquireLeadership(key1, val1, seconds10)
	require.NotNil(ctx1)

	cleanup() // cleanup => no further acquisitions

	<-ctx1.Done()
	ctx2 := elections.AcquireLeadership(key2, val2, seconds10)
	require.Nil(ctx2, "No new leadership after cleanup")
}

func cleanupDuringRenewal[K any, V any](_ *require.Assertions, elections IElections[K, V], _ ITTLStorage[K, V], cleanup func(), dataGen TestDataGen[K, V]) {
	key := dataGen.NextKey()
	val := dataGen.NextVal()
	ctx := elections.AcquireLeadership(key, val, seconds10)

	{
		testingu.MockTime.Sleep(time.Duration(seconds10/2) * time.Second)

		// now force cancel everything.
		// successful finalizing after that shows that there are no deadlocks caused by simultaneous locks in
		// cleanup() and in releaseLeadership() after leadership loss
		cleanup()
		<-ctx.Done()
	}
}
