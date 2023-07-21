/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import (
	"fmt"
	"sync"
	"testing"
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

func newCache(len int) ICache[IKey, IEvent] {
	return New[IKey, IEvent](len, func(k IKey, e IEvent) {
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

func Benchmark_CachePLogEvents(b *testing.B) {

	test := func(maxOfs uint64) {
		bomber := newBomber(0, maxOfs)

		var put, get int64
		b.Run(fmt.Sprintf("Put-%d", maxOfs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bomber.putEvents(newCache(int(maxOfs)))
			}
			put = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs)
		})

		cache := newCache(int(maxOfs))
		bomber.putEvents(cache)
		b.Run(fmt.Sprintf("Get-%d", maxOfs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bomber.getEvents(cache)
			}
			get = b.Elapsed().Nanoseconds() / int64(b.N) / int64(maxOfs)
		})
		fmt.Printf("\t— Put:\t%10d ns/op; Get:\t%10d ns/op\n", put, get)
	}

	test(100)
	test(1000)
	test(10000)
	test(100000)
	test(1000000)
}

func Benchmark_CachePLogEventsParallel(b *testing.B) {

	test := func(maxOfs uint64, bCount int) {
		bombers := make([]*bomber, bCount)
		for p := 0; p < bCount; p++ {
			bombers[p] = newBomber(uint64(p), maxOfs)
		}

		var put, get int64
		b.Run(fmt.Sprintf("Put-%d-%d", maxOfs, bCount), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				wg := sync.WaitGroup{}
				cache := newCache(int(maxOfs) * bCount)
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

		cache := newCache(int(maxOfs) * bCount)
		for p := 0; p < bCount; p++ {
			bombers[p].putEvents(cache)
		}

		b.Run(fmt.Sprintf("Get-%d-%d", maxOfs, bCount), func(b *testing.B) {
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
		fmt.Printf("\t— Put:\t%10d ns/op; Get:\t%10d ns/op\n", put, get)
	}

	test(100, 10)
	test(500, 50)
	test(1000, 100)
}
