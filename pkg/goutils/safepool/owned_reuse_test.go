/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package safepool_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/safepool"
)

type leakReportItem struct {
	safepool.IReleaser
}

// TestOwnedReusePreservesLeakReport borrows two objects from the same
// call site and returns one. Reusing that object as an owned child must
// not remove the other object's outstanding borrow from the leak report.
func TestOwnedReusePreservesLeakReport(t *testing.T) {
	require.Zero(t, safepool.GetObjectsInUse())
	safepool.SetDebug(true)
	// Cleanup callbacks run in reverse order, so debugging stays enabled
	// until the borrowed item's cleanup registered below has run.
	t.Cleanup(func() { safepool.SetDebug(false) })

	newItem := func(releaser safepool.IReleaser) *leakReportItem {
		return &leakReportItem{IReleaser: releaser}
	}
	owners := safepool.NewPool(newItem)

	// sync.Pool may discard returned objects, including under -race.
	// Retry with a fresh pool until we observe the reuse being tested.
	const maxAttempts = 32
	for range maxAttempts {
		items := safepool.NewPool(newItem)
		var borrowedItems [2]*leakReportItem
		for i := range borrowedItems {
			// Both borrows must have the same stack trace so the report
			// groups them under a single borrow site.
			borrowedItems[i] = items.Get()
		}
		borrowedItems[0].Release()

		// The borrowedItems[1] is deliberately still in use. Capture its
		// report before the returned object is borrowed as an owned child.
		var before bytes.Buffer
		safepool.PrintNonReleased(&before)
		log.Println("before owner release:\n", before.String())
		// should report "1 not released borrowed at <borrowedItems[i] = items.Get()>"
		const borrowSite = "TestOwnedReusePreservesLeakReport"
		require.Contains(
			t,
			before.String(),
			borrowSite,
			"the standalone leak must be visible before owned reuse",
		)
		require.Equal(
			t,
			uint64(1),
			safepool.GetObjectsInUse(),
			"the second standalone object is still borrowed",
		)

		// Borrow an owner. It does not borrow child items automatically.
		owner := owners.Get()

		// Try to reuse borrowedItems[0] as an owned item.
		child := items.GetOwned(owner)

		// check that we're just borrowed the previously released item
		reused := child == borrowedItems[0]

		// Release the owner and whichever child GetOwned returned.
		// The child is borrowedItems[0] only when reused is true.
		owner.Release()

		if !reused {
			// not previously released item is taken from the pool -> try again
			// that could happen sometimes, e.g. with --race
			borrowedItems[1].Release()
			continue

		}
		// Keep the outstanding borrow alive through the assertions, then
		// release it even if require stops the test on a failed assertion.
		t.Cleanup(borrowedItems[1].Release)

		// borrowedItems[1] is not released yet here !!!

		// borrowedItems[1] must remain in the leak report.
		var after bytes.Buffer
		safepool.PrintNonReleased(&after)
		log.Println("after owner release:\n", after.String())
		require.Contains(
			t,
			after.String(),
			borrowSite,
			"the standalone leak must remain visible",
		)
		return
	}

	t.Fatal("sync.Pool discarded the returned object on every attempt")
}
