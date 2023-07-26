/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package objcache_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/objcache"
)

// value with RefCounter
type value struct {
	objcache.RefCounter
	data string
}

// frees value resource data
func (s *value) Free() {
	s.data = "freed"
}

// creates new value with RefCounter
func newValue(dataSize int) *value {
	v := &value{}
	v.RefCounter.Value = v
	v.data = "allocated"
	return v
}

func Example() {
	// Create cache with size 1 to demonstrate cache value eviction
	cache := objcache.New[int64, *value](1)

	v := newValue(1024)

	// put value into cache
	{
		// reference count for new value is one
		fmt.Printf("new value      : refs: %d, data: %v\n", v.RefCount(), v.data)

		// put value to cache increase reference count
		cache.Put(1, v)
		fmt.Printf("after put      : refs: %d, data: %v\n", v.RefCount(), v.data)

		// cache.Put(1, v) — DO NOT PUT SAME VALUE TWICE! this increases ref count and avoids freeing.

		// release decrease reference count
		v.Release()
		fmt.Printf("after release 1: refs: %d, data: %v\n", v.RefCount(), v.data)
	}

	// get value from cache
	{
		v, ok := cache.Get(1)
		fmt.Println("founded        :", ok)
		fmt.Printf("after get      : refs: %d, data: %v\n", v.RefCount(), v.data)

		v.Release()
		fmt.Printf("after release 2: refs: %d, data: %v\n", v.RefCount(), v.data)
	}

	// evict value from cache
	{
		v2 := newValue(1024)
		cache.Put(2, v2)
	}

	fmt.Printf("after evicted  : refs: %d, data: %v\n", v.RefCount(), v.data)

	// Output:
	// new value      : refs: 1, data: allocated
	// after put      : refs: 2, data: allocated
	// after release 1: refs: 1, data: allocated
	// founded        : true
	// after get      : refs: 2, data: allocated
	// after release 2: refs: 1, data: allocated
	// after evicted  : refs: 0, data: freed
}
