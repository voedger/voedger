/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package pool_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/pool"
)

// cpu: AMD Ryzen 7 7700 8-Core Processor
// BenchmarkBasic/basic-16        42678502	        28.85 ns/op	       0 B/op	       0 allocs/op
func BenchmarkBasic(b *testing.B) {
	p := pool.NewPool(func(releaser pool.IReleaser) *myStruct {
		return &myStruct{IReleaser: releaser}
	})

	b.Run("basic", func(b *testing.B) {
		for range b.N {
			myStructInstance := p.Get()
			myStructInstance.Release()
		}
	})
}

type simpleStruct struct {
	pool.IReleaser
	isReleased bool
}

// cpu: AMD Ryzen 7 7700 8-Core Processor
// BenchmarkExample/pool-16       71031975	        17.25 ns/op	       0 B/op	       0 allocs/op
// BenchmarkExample/sync.Pool-16  146662080	        8.018 ns/op	       0 B/op	       0 allocs/op
func BenchmarkExample(b *testing.B) {
	p := pool.NewPool(func(releaser pool.IReleaser) *simpleStruct {
		return &simpleStruct{IReleaser: releaser}
	})

	b.Run("pool", func(b *testing.B) {
		for range b.N {
			ps := p.Get()
			ps.Release()
		}
	})

	b.Run("sync.Pool", func(b *testing.B) {
		syncPool := sync.Pool{
			New: func() interface{} {
				return &simpleStruct{}
			},
		}
		objectsInUse := uint64(0)

		b.ResetTimer()

		for range b.N {
			obj := syncPool.Get().(*simpleStruct)
			atomic.AddUint64(&objectsInUse, uint64(1))
			obj.isReleased = false

			if obj.isReleased {
				panic("already released")
			}
			syncPool.Put(obj)
			atomic.AddUint64(&objectsInUse, ^uint64(0))
		}
	})

	require.Zero(b, pool.GetObjectsInUse())
}

func BenchmarkOwned(b *testing.B) {
	b.Run("basic", func(b *testing.B) {
		for range b.N {
			owner := poolOwner.Get()
			owner.Release()
		}
	})
}
