/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/voedger/voedger/pkg/objcache"
	"github.com/voedger/voedger/pkg/objcache/internal/test"
)

func newCache(p objcache.CacheProvider, maxOfs int) test.TckCache {
	return objcache.NewProvider[test.IOffset, test.IEvent](p, maxOfs, func(o test.IOffset, _ test.IEvent) {
		panic(fmt.Errorf("unexpected event eviction at offset: %v", o))
	})
}

func sequenceBench(b *testing.B, p objcache.CacheProvider, maxOfs int) {

	bomber := test.NewBomber(0, uint64(maxOfs))

	var put, get int64
	b.Run(fmt.Sprintf("%v-Seq-Put-%d", p, maxOfs), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bomber.PutEvents(newCache(p, maxOfs))
		}
		put = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs)
	})

	cache := newCache(p, maxOfs)
	bomber.PutEvents(cache)
	b.Run(fmt.Sprintf("%v-Seq-Get-%d", p, maxOfs), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bomber.GetEvents(cache)
		}
		get = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs)
	})
	fmt.Printf("\t— %v:\t (Sequenced)\t Put:\t%10d ns/op; Get:\t%10d ns/op\n", p, put, get)
}

func parallelBench(b *testing.B, p objcache.CacheProvider, maxOfs int, bCount int) {

	bombers := make([]test.IBomber, bCount)
	for p := 0; p < bCount; p++ {
		bombers[p] = test.NewBomber(uint64(p), uint64(maxOfs))
	}

	var put, get int64
	b.Run(fmt.Sprintf("%v-Par-Put-%d-%d", p, maxOfs, bCount), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg := sync.WaitGroup{}
			cache := newCache(p, bCount*maxOfs)
			for p := 0; p < bCount; p++ {
				wg.Add(1)
				go func(p int) {
					bombers[p].PutEvents(cache)
					wg.Done()
				}(p)
			}
			wg.Wait()
		}
		put = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs) / int64(bCount)
	})

	b.Run(fmt.Sprintf("%v-Par-Get-%d-%d", p, maxOfs, bCount), func(b *testing.B) {
		cache := newCache(p, bCount*maxOfs)
		for p := 0; p < bCount; p++ {
			bombers[p].PutEvents(cache)
		}

		for i := 0; i < b.N; i++ {
			wg := sync.WaitGroup{}
			for p := 0; p < bCount; p++ {
				wg.Add(1)
				go func(p int) {
					bombers[p].GetEvents(cache)
					wg.Done()
				}(p)
			}
			wg.Wait()
		}
		get = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs) / int64(bCount)
	})
	fmt.Printf("\t— %v:\t (Parallel-%d)\t Put:\t%10d ns/op; Get:\t%10d ns/op\n", p, bCount, put, get)
}

func generalBench(b *testing.B, p objcache.CacheProvider) {
	b.Run("1. Small cache 100 events", func(b *testing.B) {
		b.Run("1.1. Sequenced", func(b *testing.B) {
			sequenceBench(b, p, 100)
		})
		b.Run("1.2. Parallel (2×50)", func(b *testing.B) {
			parallelBench(b, p, 50, 2)
		})
	})

	b.Run("2. Middle cache 1’000 events", func(b *testing.B) {
		b.Run("2.1. Sequenced", func(b *testing.B) {
			sequenceBench(b, p, 1000)
		})
		b.Run("2.2. Parallel (10×100)", func(b *testing.B) {
			parallelBench(b, p, 100, 10)
		})
	})

	b.Run("3. Big cache 10’000 events", func(b *testing.B) {
		b.Run("3.1. Sequenced", func(b *testing.B) {
			sequenceBench(b, p, 10000)
		})
		b.Run("3.2. Parallel (20×500)", func(b *testing.B) {
			parallelBench(b, p, 500, 20)
		})
	})

	b.Run("4. Large cache 100’000 events", func(b *testing.B) {
		b.Run("3.1. Sequenced", func(b *testing.B) {
			sequenceBench(b, p, 100000)
		})
		b.Run("3.2. Parallel (100×1000)", func(b *testing.B) {
			parallelBench(b, p, 1000, 100)
		})
	})
}

func BenchmarkCacheGeneralHashicorp(b *testing.B) {
	generalBench(b, objcache.Hashicorp)
}

func BenchmarkCacheGeneralTheine(b *testing.B) {
	generalBench(b, objcache.Theine)
}

func BenchmarkCacheGeneralFloatdrop(b *testing.B) {
	generalBench(b, objcache.Floatdrop)
}

func BenchmarkCacheGeneralImcache(b *testing.B) {
	generalBench(b, objcache.Imcache)
}

func BenchmarkCacheParallelHashicorp(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		parallelBench(b, objcache.Hashicorp, maxOfs, part)
	}
}

func BenchmarkCacheParallelTheine(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		parallelBench(b, objcache.Theine, maxOfs, part)
	}
}

func BenchmarkCacheParallelFloatdrop(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		parallelBench(b, objcache.Floatdrop, maxOfs, part)
	}
}

func BenchmarkCacheParallelImcache(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		parallelBench(b, objcache.Imcache, maxOfs, part)
	}
}
