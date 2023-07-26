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
//  1. Client cache value should has Free() method.
//  2. Item struct should be included into client value and Item.Value field must be assigned to client value.
//
// Cache increments reference counter then you put value into cache and
// then you get value from it. So, you do not need to call AddRef() manually.
//
// Every time then you finish use value you should call Release(), this
// decrement reference counter. If value evicted from cache, then cache calls
// Release() too. When reference counter decreases to zero, Free()
// method of value will be called.
type Item struct {
	count atomic.Int32
	Value interface{ Free() }
}

// Increases reference count by 1. Return false if reference count is zero
// and item value is about released
func (i *Item) AddRef() bool {
	for cnt := i.count.Load(); cnt >= 0; cnt = i.count.Load() {
		if new := cnt + 1; i.count.CompareAndSwap(cnt, new) {
			return true
		}
	}
	// notest
	return false
}

// Returns current reference count
func (i *Item) RefCount() int {
	return int(i.count.Load() + 1)
}

// Decrease reference count by 1. If counter decreases to zero then calls
// item value Free() method
func (i *Item) Release() {
	for cnt := i.count.Load(); cnt >= 0; cnt = i.count.Load() {
		if new := cnt - 1; i.count.CompareAndSwap(cnt, new) {
			if new == -1 {
				if i.Value != nil {
					i.Value.Free()
				}
			}
			break
		}
	}
}
