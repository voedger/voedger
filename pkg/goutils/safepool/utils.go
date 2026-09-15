/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package safepool

import (
	"fmt"
	"io"
	"strings"
)

// GetObjectsInUse returns total amount of objects taken from all pools but not returned
// useful in tests
func GetObjectsInUse() uint64 {
	m.Lock()
	// Registrations only append; existing entries never change, so copying
	// the slice header is enough to safely iterate after unlocking.
	counters := objectsCounters
	m.Unlock()

	// Callbacks may call other pool functions that acquire the same mutex.
	res := uint64(0)
	for _, oc := range counters {
		res += oc()
	}
	return res
}

// RegisterObjectsInUseCounter adds a counter to GetObjectsInUse.
// NewPool calls it automatically for every pool it creates. Register counters from
// other pool implementations to include them in the shared total.
// The counter must be thread-safe.
func RegisterObjectsInUseCounter(oc func() uint64) {
	m.Lock()
	objectsCounters = append(objectsCounters, oc)
	m.Unlock()
}

// PrintNonReleased prints stacktraces that explains where non-released objects were borrowed
// Debug mode must be enabled with safepool.SetDebug(true).
func PrintNonReleased(w io.Writer) {
	nr := getNonReleased()
	if len(nr) == 0 {
		return
	}
	fmt.Fprintln(w, "objects borrowed from pools but not released:")
	for st, amount := range nr {
		st = "\t" + strings.ReplaceAll(st, "\n", "\n\t")
		st = st[:len(st)-1]
		fmt.Fprintf(w, "%d not released borrowed at:\n%s", amount, st)
	}
}

// SetDebug switches debug mode. In debug mode pool engine tracks amounts of non-released objects
// per each borrow source code point (for all pools)
// use PrintNonReleased() to get explanations
// useful for investigations only, decreases performance
func SetDebug(enabled bool) {
	isDebug.Store(enabled)
}

func getNonReleased() map[string]int {
	m.Lock()
	res := map[string]int{}
	for k, v := range objAmounts {
		if v > 0 {
			res[k] = v
		}
	}
	m.Unlock()
	return res
}
