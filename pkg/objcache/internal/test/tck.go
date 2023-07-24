/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type NewTckCache func(size int, onEvicted func(IOffset, IEvent)) TckCache

func TechnologyCompatibilityKit(t *testing.T, new NewTckCache) {
	require := require.New(t)

	const (
		bombersCount    = 100
		eventsPerBomber = 1000
		cacheSize       = bombersCount * eventsPerBomber //100’000
	)

	bombers := make([]IBomber, bombersCount)
	for p := 0; p < bombersCount; p++ {
		bombers[p] = NewBomber(uint64(p), uint64(eventsPerBomber))
	}

	cache := new(cacheSize, func(o IOffset, _ IEvent) {
		require.Fail("unexpected eviction", "offset: %v", o)
	})

	t.Run("must be ok to put events into cache", func(t *testing.T) {
		wg := sync.WaitGroup{}
		for p := 0; p < bombersCount; p++ {
			wg.Add(1)
			go func(p int) {
				bombers[p].PutEvents(cache)
				wg.Done()
			}(p)
		}
		wg.Wait()
	})

	t.Run("must be ok to get events from cache", func(t *testing.T) {
		wg := sync.WaitGroup{}
		for p := 0; p < bombersCount; p++ {
			wg.Add(1)
			go func(p int) {
				bombers[p].GetEvents(cache)
				wg.Done()
			}(p)
		}
		wg.Wait()
	})

	t.Run("must be ok to evict events from cache", func(t *testing.T) {
		const cacheSize = 10

		evicted := make(map[IOffset]IEvent)

		cache := new(cacheSize, func(o IOffset, e IEvent) {
			evicted[o] = e
		})

		bomber := NewBomber(1, 2*cacheSize)
		bomber.PutEvents(cache)

		for len(evicted) < cacheSize {
			time.Sleep(time.Nanosecond)
		}

		require.Len(evicted, cacheSize)
	})
}
