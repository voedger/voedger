/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import (
	"sync/atomic"
)

// Client cache values can include this structure to automate value references
// and value releasing.
//
// # Automation:
//
// Cache increments reference counter then you put value into cache and
// then you get value from it. So, you do not need to call AddRef() manually.
//
// Every time then you finish use value you should call Release(), this
// decrement reference counter. If value evicted from cache, then cache calls
// Release() too. When reference counter decreases to zero, Released()
// method of value will be called.
type Ref struct {
	count atomic.Int32
	Value interface{ Released() }
}

// Increases reference count by 1. Return false if reference count is zero
// and value is about released
func (r *Ref) AddRef() bool {
	for ref := r.count.Load(); ref >= 0; ref = r.count.Load() {
		if new := ref + 1; r.count.CompareAndSwap(ref, new) {
			return true
		}
	}
	// notest
	return false
}

// Returns current reference count
func (r *Ref) RefCount() int {
	return int(r.count.Load() + 1)
}

// Decrease reference count by 1. If counter decreases to zero then calls
// value Released() method
func (r *Ref) Release() {
	for ref := r.count.Load(); ref >= 0; ref = r.count.Load() {
		if new := ref - 1; r.count.CompareAndSwap(ref, new) {
			if new == -1 {
				if r.Value != nil {
					r.Value.Released()
				}
			}
			break
		}
	}
}
