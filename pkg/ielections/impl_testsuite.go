/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

const seconds10 = 10

func ElectionsTestSuite(t *testing.T, ttlStorage ITTLStorage[string, string]) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()
	tests := []func(*require.Assertions, IElections[string, string], ITTLStorage[string, string], func()){
		basicUsage,
		acquireLeadershipIfAcquiredAlready,
		closeContextOnCompareAndSwapFailure_KeyChanged,
		closeContextOnCompareAndSwapFailure_KeyDeleted,
		releaseLeadershipWithoutAcquire,
		acquireFailingAfterCleanup,
		testCleanupDuringRenewal,
	}
	// note: testing the case when ttl record is expired is nonsense because it is renewing, can not be expired. Key deletion case is covered
	for _, test := range tests {
		name := runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
		name = name[strings.LastIndex(name, ".")+1:]
		t.Run(name, func(t *testing.T) {
			elections, cleanup := Provide(ttlStorage, coreutils.MockTime)
			defer cleanup()
			require := require.New(t)
			test(require, elections, ttlStorage, cleanup)
		})
	}
}

func basicUsage(require *require.Assertions, elections IElections[string, string], iTTLStorage ITTLStorage[string, string], _ func()) {
	ctx := elections.AcquireLeadership("testKey", "leaderVal", seconds10)
	require.NotNil(ctx, "Should return a non-nil context on successful acquisition")

	ok, storedData, err := iTTLStorage.Get("testKey")
	require.NoError(err)
	require.True(ok)
	require.Equal("leaderVal", storedData)

	// Release leadership
	elections.ReleaseLeadership("testKey")
	<-ctx.Done()
}

func acquireLeadershipIfAcquiredAlready(require *require.Assertions, elections IElections[string, string], iTTLStorage ITTLStorage[string, string], _ func()) {
	ok, err := iTTLStorage.InsertIfNotExist("occupied", "someVal", seconds10)
	require.NoError(err)
	require.True(ok)
	ctx := elections.AcquireLeadership("occupied", "myVal", seconds10)
	require.Nil(ctx, "Should return nil if the key is already in storage")
}

// func testElections_AcquireLeadership_Error(t *testing.T, elections IElections[string, string], iTTLStorage ITTLStorage[string, string]) {
// 	iTTLStorage.errorTrigger["badKey"] = true
// 	ctx := elector.AcquireLeadership("badKey", "val", seconds10)
// 	require.Nil(t, ctx, "Should return nil if a storage error occurs")
// }

func closeContextOnCompareAndSwapFailure_KeyChanged(require *require.Assertions, elections IElections[string, string], iTTLStorage ITTLStorage[string, string], _ func()) {
	ctx := elections.AcquireLeadership("renewKey", "renewVal", seconds10)
	require.NotNil(ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	ok, err := iTTLStorage.CompareAndSwap("renewKey", "renewVal", "someOtherVal", seconds10*2)
	require.NoError(err)
	require.True(ok)

	// trigger the renewal
	coreutils.MockTime.Sleep((seconds10) * time.Second)

	// The leadership is forcibly released in the background.
	<-ctx.Done()

	// expect that the new value is still stored in the storage
	ok, keptValue, err := iTTLStorage.Get("renewKey")
	require.NoError(err)
	require.True(ok)
	require.Equal("someOtherVal", keptValue)

	// make the sabotaged key be expired
	coreutils.MockTime.Sleep((seconds10 + 1) * time.Second)
}

func closeContextOnCompareAndSwapFailure_KeyDeleted(require *require.Assertions, elections IElections[string, string], iTTLStorage ITTLStorage[string, string], _ func()) {
	ctx := elections.AcquireLeadership("renewKey", "renewVal", seconds10)
	require.NotNil(ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	ok, err := iTTLStorage.CompareAndDelete("renewKey", "renewVal")
	require.NoError(err)
	require.True(ok)

	// trigger the renewal
	coreutils.MockTime.Sleep((seconds10 + 1) * time.Second)

	// The leadership is forcibly released in the background.
	<-ctx.Done()

	// just check the key is deleted indeed
	ok, _, err = iTTLStorage.Get("renewKey")
	require.NoError(err)
	require.False(ok)
}

func releaseLeadershipWithoutAcquire(require *require.Assertions, elections IElections[string, string], iTTLStorage ITTLStorage[string, string], _ func()) {
	// Releasing unknown key => nothing happens, no errors
	elections.ReleaseLeadership("unknownKey")
}

func acquireFailingAfterCleanup(require *require.Assertions, elections IElections[string, string], iTTLStorage ITTLStorage[string, string], cleanup func()) {
	ctx1 := elections.AcquireLeadership("keyA", "valA", seconds10)
	require.NotNil(ctx1)

	cleanup() // cleanup => no further acquisitions

	<-ctx1.Done()
	ctx2 := elections.AcquireLeadership("keyB", "valB", seconds10)
	require.Nil(ctx2, "No new leadership after cleanup")
}

func testCleanupDuringRenewal(_ *require.Assertions, elections IElections[string, string], _ ITTLStorage[string, string], cleanup func()) {
	// compareAndSwapCalled := make(chan interface{})
	// iTTLStorage.onBeforeCompareAndSwap = func() {
	// 	close(compareAndSwapCalled)
	// }

	ctx := elections.AcquireLeadership("expireKey", "expireVal", seconds10)

	{
		coreutils.MockTime.Sleep(time.Duration(seconds10/2) * time.Second)

		// gauarantee that <-ticker case is fired, not <-li.ctx.Done()
		// otherwise we do not know which case will fire on cleanup()
		// <-compareAndSwapCalled

		// now force cancel everything.
		// successful finalizing after that shows that there are no deadlocks caused by simultaneous locks in
		// cleanup() and in releaseLeadership() after leadership loss
		cleanup()
		<-ctx.Done()
	}
}
