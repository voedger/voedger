/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package pool_test

import (
	"bytes"
	"io"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/bytebufferpool"
	"github.com/voedger/voedger/pkg/goutils/pool"
)

type myStruct struct {
	// each pooled struct must include IReleaser field that provides Release() ability.
	// this field will initialized in the instantiator
	pool.IReleaser

	// example nested field that requires personal handling (e.g. borrow\release)
	bb   *bytebufferpool.ByteBuffer
	fld1 int
}

// optional Clenaup() will be called automatically right before returning the myStruct instance to the pool
func (ms *myStruct) Cleanup() {
	bytebufferpool.Put(ms.bb)
	ms.bb = nil
}

// optional Init() will be called automatically on each myStruct instance borrow. It should init the current instance
func (ms *myStruct) Init() {
	ms.bb = bytebufferpool.Get()
}

func TestBasicUsage_Simple(t *testing.T) {
	require := require.New(t)
	p := pool.NewPool(func(releaser pool.IReleaser) *myStruct {
		// instantiator must manually initialize IReleaser field with the provided implementation
		return &myStruct{IReleaser: releaser}
	})

	// borrow an instance of *myStruct
	myStructInstance := p.Get()

	// internal initialization is done in myStruct.Init()
	require.NotNil(myStructInstance.bb)

	// 1 object in use
	require.Equal(uint64(1), pool.GetObjectsInUse())

	// return the instance back to the pool
	myStructInstance.Release()
	// myStruct.bb is automatically returned back to `bytebufferpool` by myStruct.Cleanup()
	// myStructInstance is returned to the pool
	// myStructInstance as well as its any member must not be used (even touched) from now on

	// unable to return the same object to the pool twice
	require.Panics(func() { myStructInstance.Release() })

	// no objects in use
	require.Zero(pool.GetObjectsInUse())
}

type referenceCountItem struct {
	pool.IReleaser
	cleanup func()
}

func (i *referenceCountItem) Cleanup() {
	i.cleanup()
}

func TestReferenceCount(t *testing.T) {
	require := require.New(t)
	objectsBefore := pool.GetObjectsInUse()
	pool.SetDebug(true)
	t.Cleanup(func() { pool.SetDebug(false) })
	var cleanupCalls atomic.Int32
	items := pool.NewPool(func(releaser pool.IReleaser) *referenceCountItem {
		return &referenceCountItem{
			IReleaser: releaser,
			cleanup:   func() { cleanupCalls.Add(1) },
		}
	})

	item := items.Get()
	item.AddRef()

	item.Release()
	var whileReferenced bytes.Buffer
	pool.PrintNonReleased(&whileReferenced)
	require.Equal(objectsBefore+1, pool.GetObjectsInUse())
	require.Zero(cleanupCalls.Load(), "a remaining reference keeps the object borrowed")
	require.Contains(whileReferenced.String(), "TestReferenceCount")

	item.Release()
	var afterFinalRelease bytes.Buffer
	pool.PrintNonReleased(&afterFinalRelease)
	require.Equal(objectsBefore, pool.GetObjectsInUse())
	require.Equal(int32(1), cleanupCalls.Load(), "the final reference releases the object")
	require.Empty(afterFinalRelease.String())
	require.PanicsWithValue("already released", func() { item.Release() })
	require.PanicsWithValue("already released", func() { item.AddRef() })
}

func TestConcurrentReferenceRelease(t *testing.T) {
	require := require.New(t)
	objectsBefore := pool.GetObjectsInUse()
	var cleanupCalls atomic.Int32
	items := pool.NewPool(func(releaser pool.IReleaser) *referenceCountItem {
		return &referenceCountItem{
			IReleaser: releaser,
			cleanup:   func() { cleanupCalls.Add(1) },
		}
	})
	item := items.Get()
	item.AddRef()

	start := make(chan struct{})
	results := make(chan any, 2)
	for range 2 {
		go func() {
			<-start
			func() {
				defer func() { results <- recover() }()
				item.Release()
			}()
		}()
	}
	close(start)

	require.Nil(<-results)
	require.Nil(<-results)
	require.Equal(int32(1), cleanupCalls.Load())
	require.Equal(objectsBefore, pool.GetObjectsInUse())
}

func TestObjectsUsageTrackInDebugMode(t *testing.T) {
	require := require.New(t)
	pool.SetDebug(true)
	defer pool.SetDebug(false)
	p := pool.NewPool(func(releaser pool.IReleaser) *myStruct {
		return &myStruct{IReleaser: releaser}
	})

	// borrow 10 instances
	roots := []*myStruct{}
	for range 10 {
		roots = append(roots, p.Get())
	}

	// one more as an example
	roots = append(roots, p.Get())

	// release one as an example
	roots[5].Release()

	// prints code points where objects were borrowed but not released
	pool.PrintNonReleased(os.Stdout)

	for i, root := range roots {
		if i != 5 {
			root.Release()
		}
	}

	// prints nothing
	pool.PrintNonReleased(os.Stdout)

	require.Zero(pool.GetObjectsInUse())
}

func TestStub(t *testing.T) {
	require := require.New(t)
	poolOwner := pool.NewPoolStub(func(releaser pool.IReleaser) *owner {
		return &owner{
			IReleaser: releaser,
		}
	})
	originalPoolNested := poolNested
	// Restore the real pool for later tests and benchmarks, even if an
	// assertion fails while this test is using the stub.
	t.Cleanup(func() { poolNested = originalPoolNested })
	poolNested = pool.NewPoolStub(func(releaser pool.IReleaser) *nested {
		return &nested{
			IReleaser: releaser,
		}
	})

	// borrow a struct, initialize fields
	owner := poolOwner.Get()
	require.Equal(uint64(3), pool.GetObjectsInUse())

	// owned struct can not be accidentally released before owner
	require.Panics(func() { owner.nested.Release() })

	// owner will release its internal fields and an owned struct using its special releaser
	// after that `owner` struct itself will be returned to the pool engine
	owner.Release()

	// unable to release twice in stub mode as well to avoid cleanup() unexpected execution
	require.Panics(func() { owner.Release() })

	require.Zero(pool.GetObjectsInUse())
}

func TestStress(t *testing.T) {
	p := pool.NewPool(func(releaser pool.IReleaser) *myStruct { return &myStruct{IReleaser: releaser} })
	ch := make(chan *myStruct)
	nch := make(chan int, 1000)
	for i := range 1000 {
		go func(i int) {
			ts1 := p.Get()
			ts1.fld1 = i
			ch <- ts1
		}(i)
		go func() {
			obj := <-ch
			n := obj.fld1
			obj.Release()
			nch <- n
		}()
	}

	numbers := map[int]struct{}{}
	for range 1000 {
		n := <-nch
		require.Less(t, n, 1000, n)
		if _, exists := numbers[n]; exists {
			t.Fatal()
		}
		numbers[n] = struct{}{}
	}
	require.Zero(t, pool.GetObjectsInUse())
}

// TestCounterCallbackCanPrintLeaks registers a counter that prints leak
// diagnostics before returning its count. GetObjectsInUse should call
// that counter, receive its result, and finish without getting stuck.
func TestCounterCallbackCanPrintLeaks(t *testing.T) {
	pool.SetDebug(true)
	defer pool.SetDebug(false)
	// Pretend an external pool has one object in use. Use an atomic counter
	// because registered callbacks must be safe to call concurrently.
	var externalObjectsInUse atomic.Uint64
	externalObjectsInUse.Store(1)
	// The counter stays registered after this test, so return its count to
	// zero during cleanup to avoid affecting later tests.
	t.Cleanup(func() { externalObjectsInUse.Store(0) })

	// Registration saves the callback; it does not call it yet.
	pool.RegisterObjectsInUseCounter(func() uint64 {
		// GetObjectsInUse is now calling our counter. Before returning
		// the count, request a report of unreleased pooled objects.
		// Discard the report text; we only need this call to finish.
		pool.PrintNonReleased(io.Discard)
		return externalObjectsInUse.Load()
	})

	// Before the fix, GetObjectsInUse held the lock while calling our external counter.
	// The counter called PrintNonReleased, which tried to acquire the same lock -> stuck.
	// GetObjectsInUse must release the lock before calling counters so this check can finish.
	require.Equal(t, uint64(1), pool.GetObjectsInUse())
}

// An application object gets Release from IReleaser and supplies its own
// Cleanup hook, just as it would when using the pool outside this test.
type concurrentReleaseItem struct {
	pool.IReleaser
	cleanup func()
}

func (i *concurrentReleaseItem) Cleanup() {
	i.cleanup()
}

// TestConcurrentRelease pauses the first Release during cleanup, then
// calls Release again on the same object. The second call must panic,
// preventing duplicate cleanup and an incorrect usage count.
func TestConcurrentRelease(t *testing.T) {
	cleanupStarted := make(chan struct{})
	continueCleanup := make(chan struct{})
	// Cleanup can be entered by both callers, so count calls atomically.
	var cleanupCalls atomic.Int32
	p := pool.NewPool(func(releaser pool.IReleaser) *concurrentReleaseItem {
		return &concurrentReleaseItem{
			IReleaser: releaser,
			cleanup: func() {
				// Pause only the first cleanup. If the duplicate release
				// reaches this hook, let it finish so we can observe the bug.
				if cleanupCalls.Add(1) == 1 {
					close(cleanupStarted)
					<-continueCleanup
				}
			},
		}
	})
	obj := p.Get()

	// Recover in the releasing goroutine and send its result back for
	// assertions in the test goroutine. A nil result means no panic.
	firstReleaseDone := make(chan any, 1)
	go func() {
		defer func() { firstReleaseDone <- recover() }()
		obj.Release()
	}()

	// The channel guarantees that the first Release is paused inside
	// Cleanup before we attempt the second Release on the same object.
	<-cleanupStarted
	var secondPanic any
	func() {
		defer func() { secondPanic = recover() }()
		obj.Release()
	}()
	// Resume the first release and collect its result before using require:
	// a failed assertion must not leave that goroutine blocked in Cleanup.
	close(continueCleanup)
	firstPanic := <-firstReleaseDone

	require.Nil(t, firstPanic, "the first release must succeed")
	// The duplicate must be rejected even while the first call is in progress.
	require.Equal(t, "already released", secondPanic, "an overlapping release must be rejected")
	require.Equal(t, int32(1), cleanupCalls.Load(), "cleanup must run once per borrow")
	// One successful release balances the single Get. If both releases
	// decrement the counter, it underflows instead of returning to zero.
	require.Zero(t, pool.GetObjectsInUse(), "one borrow must be counted as released exactly once")
}

// TestReleaseClearsLeakReportAfterDebugDisabled borrows with debug mode
// enabled, then disables it before releasing. The released object must
// disappear from the report, including after debug mode is enabled again.
func TestReleaseClearsLeakReportAfterDebugDisabled(t *testing.T) {
	type debugModeItem struct {
		pool.IReleaser
	}
	for _, tc := range []struct {
		name    string
		newPool func(func(pool.IReleaser) *debugModeItem) pool.IPool[*debugModeItem]
	}{
		{name: "normal", newPool: pool.NewPool[*debugModeItem]},
		{name: "stub", newPool: pool.NewPoolStub[*debugModeItem]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool.SetDebug(true)
			t.Cleanup(func() { pool.SetDebug(false) })
			items := tc.newPool(func(releaser pool.IReleaser) *debugModeItem {
				return &debugModeItem{IReleaser: releaser}
			})

			// Get records this borrow because debugging is enabled.
			item := items.Get()
			var before bytes.Buffer
			pool.PrintNonReleased(&before)

			// Stop recording new borrows, then release the tracked object.
			// Its recorded trace must still be removed during release.
			pool.SetDebug(false)
			item.Release()
			var whileDisabled bytes.Buffer
			pool.PrintNonReleased(&whileDisabled)

			// Enabling diagnostics again must not reveal a phantom leak.
			pool.SetDebug(true)
			var after bytes.Buffer
			pool.PrintNonReleased(&after)

			// Assert after release so a failure cannot leave the object borrowed.
			require.Contains(t, before.String(), "TestReleaseClearsLeakReportAfterDebugDisabled")
			require.Contains(t, before.String(), "1 not released borrowed at:")
			require.Zero(t, pool.GetObjectsInUse(), "the object was released")
			require.Empty(t, after.String(), "enabling debug again must not report a released object")
			require.Empty(t, whileDisabled.String(), "release must remove the trace even with debug disabled")
		})
	}
}

type initPanicItem struct {
	pool.IReleaser
	initialize func(*initPanicItem)
}

func (i *initPanicItem) Init() {
	if i.initialize != nil {
		i.initialize(i)
	}
}

// TestGetTracksInitPanic checks that an object taken from its pool remains
// counted and reported when Init panics before Get can return it. An owned
// child borrowed by Init must be counted in addition to the failed owner.
func TestGetTracksInitPanic(t *testing.T) {
	for _, mode := range []struct {
		name    string
		newPool func(func(pool.IReleaser) *initPanicItem) pool.IPool[*initPanicItem]
	}{
		{name: "normal", newPool: pool.NewPool[*initPanicItem]},
		{name: "stub", newPool: pool.NewPoolStub[*initPanicItem]},
	} {
		for _, withChild := range []bool{false, true} {
			scenario := "standalone"
			if withChild {
				scenario = "with_owned_child"
			}
			t.Run(mode.name+"/"+scenario, func(t *testing.T) {
				// Record the global count to check this borrow and its cleanup.
				beforeCount := pool.GetObjectsInUse()
				pool.SetDebug(true)
				t.Cleanup(func() { pool.SetDebug(false) })

				var children pool.IPool[*initPanicItem]
				wantCount := uint64(1)
				if withChild {
					children = mode.newPool(func(releaser pool.IReleaser) *initPanicItem {
						return &initPanicItem{IReleaser: releaser, initialize: nil}
					})
					wantCount++
				}

				// Keep the factory-created object only for test cleanup.
				// A normal caller cannot obtain it from Get after the panic.
				var created *initPanicItem
				items := mode.newPool(func(releaser pool.IReleaser) *initPanicItem {
					created = &initPanicItem{
						IReleaser: releaser,
						initialize: func(item *initPanicItem) {
							if children != nil {
								// This child is now tied to the owner's lifetime.
								children.GetOwned(item)
							}
							panic("init failed")
						},
					}
					return created
				})

				// Recover the expected panic, as an application might at a
				// request boundary, then inspect diagnostics through the API.
				require.PanicsWithValue(t, "init failed", func() { items.Get() })
				var report bytes.Buffer
				pool.PrintNonReleased(&report)
				borrowedCount := pool.GetObjectsInUse() - beforeCount

				// A failed Init leaves the borrow tracked until explicitly released.
				// Release the captured object and its children before checking the
				// saved diagnostics, so a failed assertion cannot leave them borrowed.
				created.Release()
				var afterRelease bytes.Buffer
				pool.PrintNonReleased(&afterRelease)

				require.Equal(t, wantCount, borrowedCount, "the failed owner must also be counted")
				require.Equal(t, beforeCount, pool.GetObjectsInUse())
				require.Contains(t, report.String(), "TestGetTracksInitPanic")
				require.Contains(t, report.String(), "1 not released borrowed at:")
				require.Empty(t, afterRelease.String(), "release must remove the failed borrow's trace")
			})
		}
	}
}
