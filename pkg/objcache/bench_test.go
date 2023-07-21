/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

type (
	IKey = interface {
		Part() uint64
		Ofs() uint64
	}

	IEvent = interface {
		Part() uint64
		Ofs() uint64
	}

	key struct {
		part uint64
		ofs  uint64
	}

	event struct {
		part uint64
		ofs  uint64
		data [1024]byte
	}

	events map[IKey]IEvent
)

func (k *key) Part() uint64   { return k.part }
func (k *key) Ofs() uint64    { return k.ofs }
func (e *event) Part() uint64 { return e.part }
func (e *event) Ofs() uint64  { return e.ofs }

func newEvents(part uint64, maxOfs uint64) events {
	events := make(map[IKey]IEvent)
	for o := uint64(0); o < maxOfs; o++ {
		k := &key{part, o}
		e := &event{part: part, ofs: o}
		events[k] = e
	}
	return events
}

func newCache(p CacheProvider, len int) ICache[IKey, IEvent] {
	return NewProvider[IKey, IEvent](p, len, func(k IKey, e IEvent) {
		panic(fmt.Errorf("unexpected event eviction, partition:%v, offset: %v", k.Part(), k.Ofs()))
	})
}

type bomber struct {
	part uint64
	events
}

func newBomber(part uint64, maxOfs uint64) *bomber {
	return &bomber{
		part:   part,
		events: newEvents(part, maxOfs),
	}
}

func (b *bomber) putEvents(cache ICache[IKey, IEvent]) {
	for k, e := range b.events {
		cache.Put(k, e)
	}
}

func (b *bomber) getEvents(cache ICache[IKey, IEvent]) {
	for k := range b.events {
		_, ok := cache.Get(k)
		if !ok {
			panic(fmt.Errorf("missed event in cache, partition:%v, offset: %v", k.Part(), k.Ofs()))
		}
	}
}

func sequenceBench(b *testing.B, p CacheProvider, maxOfs int) {

	bomber := newBomber(0, uint64(maxOfs))

	var put, get int64
	b.Run(fmt.Sprintf("%v-Seq-Put-%d", p, maxOfs), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bomber.putEvents(newCache(p, int(maxOfs)))
		}
		put = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs)
	})

	cache := newCache(p, int(maxOfs))
	bomber.putEvents(cache)
	b.Run(fmt.Sprintf("%v-Seq-Get-%d", p, maxOfs), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bomber.getEvents(cache)
		}
		get = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs)
	})
	fmt.Printf("\t— %v:\t (Sequenced)\t Put:\t%10d ns/op; Get:\t%10d ns/op\n", p, put, get)
}

func parallelBench(b *testing.B, p CacheProvider, maxOfs int, bCount int) {

	bombers := make([]*bomber, bCount)
	for p := 0; p < bCount; p++ {
		bombers[p] = newBomber(uint64(p), uint64(maxOfs))
	}

	var put, get int64
	b.Run(fmt.Sprintf("%v-Par-Put-%d-%d", p, maxOfs, bCount), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg := sync.WaitGroup{}
			cache := newCache(p, int(maxOfs)*bCount)
			for p := 0; p < bCount; p++ {
				wg.Add(1)
				go func(p int) {
					bombers[p].putEvents(cache)
					wg.Done()
				}(p)
			}
			wg.Wait()
		}
		put = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs) / int64(bCount)
	})

	cache := newCache(p, int(maxOfs)*bCount)
	for p := 0; p < bCount; p++ {
		bombers[p].putEvents(cache)
	}

	b.Run(fmt.Sprintf("%v-Par-Get-%d-%d", p, maxOfs, bCount), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg := sync.WaitGroup{}
			for p := 0; p < bCount; p++ {
				wg.Add(1)
				go func(p int) {
					bombers[p].getEvents(cache)
					wg.Done()
				}(p)
			}
			wg.Wait()
		}
		get = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs) / int64(bCount)
	})
	fmt.Printf("\t— %v:\t (Parallel-%d)\t Put:\t%10d ns/op; Get:\t%10d ns/op\n", p, bCount, put, get)
}

func generalBench(b *testing.B, p CacheProvider) {
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
	generalBench(b, Hashicorp)
}

func BenchmarkCacheGeneralTheine(b *testing.B) {
	generalBench(b, Theine)
}

func BenchmarkCacheGeneralFloatdrop(b *testing.B) {
	generalBench(b, Floatdrop)
}

func BenchmarkCacheGeneralImcache(b *testing.B) {
	generalBench(b, Imcache)
}

func BenchmarkCacheParallelismHashicorp(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		runtime.GC()
		time.Sleep(time.Second)
		parallelBench(b, Hashicorp, maxOfs, part)
	}
}

func BenchmarkCacheParallelismTheine(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		runtime.GC()
		time.Sleep(time.Second)
		parallelBench(b, Theine, maxOfs, part)
	}
}

func BenchmarkCacheParallelismFloatdrop(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		runtime.GC()
		time.Sleep(time.Second)
		parallelBench(b, Floatdrop, maxOfs, part)
	}
}

func BenchmarkCacheParallelismImcache(b *testing.B) {
	const (
		maxOfs  = 1000
		maxPart = 101
	)
	for part := 1; part <= maxPart; part += 10 {
		runtime.GC()
		time.Sleep(time.Second)
		parallelBench(b, Imcache, maxOfs, part)
	}
}
