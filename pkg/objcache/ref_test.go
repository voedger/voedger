/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/objcache"
)

// value with Ref
type value struct {
	objcache.Ref
	data string
}

// releases value resource data
func (s *value) Released() {
	s.data = "released"
}

// creates new value with Ref
func newValue(dataSize int) *value {
	v := &value{}
	v.Value = v
	v.data = "assigned"
	return v
}

func ExampleRef() {
	cache := objcache.New[int64, *value](1)

	v := newValue(1024)

	// put value into cache
	{
		// reference count for new value is one
		fmt.Printf("new value      : refs: %d, data: %v\n", v.RefCount(), v.data)

		// put value to cache increase reference count
		cache.Put(1, v)
		fmt.Printf("after put      : refs: %d, data: %v\n", v.RefCount(), v.data)

		// release decrease reference count
		v.Release()
		fmt.Printf("after release 1: refs: %d, data: %v\n", v.RefCount(), v.data)
	}

	// get value from cache
	{
		v, ok := cache.Get(1)
		fmt.Println(ok)
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
	// new value      : refs: 1, data: assigned
	// after put      : refs: 2, data: assigned
	// after release 1: refs: 1, data: assigned
	// true
	// after get      : refs: 2, data: assigned
	// after release 2: refs: 1, data: assigned
	// after evicted  : refs: 0, data: released
}
