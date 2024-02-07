/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package objcache

import (
	"sync/atomic"
)

// Client cache values can include RefCounter to automate value references
// and value freeing.
//
// # Automation:
//  1. Client cache value should has Free() method.
//  2. RefCounter struct should be included into client value and RefCounter.Value
//     field must be assigned to client value.
//
// Cache increments reference counter then you put value into cache and
// then you get value from it. So, you do not need to call AddRef() manually.
//
// Every time then you finish use value you should call Release(), this
// decrement reference counter. If value evicted from cache, then cache calls
// Release() too. When reference counter decreases to zero, Free()
// method of value will be called.
type RefCounter struct {
	count atomic.Int32
	Value interface{ Free() }
}

// Increases reference count by 1.
//
// # Panics
// - if reference count is zero and item value is about released
func (rc *RefCounter) AddRef() {
	if rc.tryAddRef() {
		return
	}
	panic(ErrRefCountIsZero)
}

// Returns current reference count
func (rc *RefCounter) RefCount() int {
	return int(rc.count.Load() + 1)
}

// Decreases reference count by 1. If counter decreases to zero, then calls
// item value Free() method
func (rc *RefCounter) Release() {
	for cnt := rc.count.Load(); cnt >= 0; cnt = rc.count.Load() {
		if newCnt := cnt - 1; rc.count.CompareAndSwap(cnt, newCnt) {
			if newCnt == -1 {
				if rc.Value != nil {
					rc.Value.Free()
				}
			}
			break
		}
	}
}

// Tries to increase reference count by 1. Return false if reference count is zero
// and item value is about released
func (rc *RefCounter) tryAddRef() bool {
	for cnt := rc.count.Load(); cnt >= 0; cnt = rc.count.Load() {
		if newCnt := cnt + 1; rc.count.CompareAndSwap(cnt, newCnt) {
			return true
		}
	}
	// notest
	return false
}
