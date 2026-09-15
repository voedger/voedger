/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package safepool_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/safepool"
)

type concurrentOwner struct {
	safepool.IReleaser
}

type concurrentOwnedItem struct {
	safepool.IReleaser
	cleanupCalls *atomic.Int32
}

func (i *concurrentOwnedItem) Cleanup() {
	i.cleanupCalls.Add(1)
}

func newConcurrentOwnershipPools(
	cleanupCalls *atomic.Int32,
) (
	owners safepool.IPool[*concurrentOwner],
	items safepool.IPool[*concurrentOwnedItem],
) {
	owners = safepool.NewPool(func(releaser safepool.IReleaser) *concurrentOwner {
		return &concurrentOwner{IReleaser: releaser}
	})
	items = safepool.NewPool(func(releaser safepool.IReleaser) *concurrentOwnedItem {
		return &concurrentOwnedItem{
			IReleaser:    releaser,
			cleanupCalls: cleanupCalls,
		}
	})
	return owners, items
}

func TestConcurrentGetOwnedPreservesEveryItem(t *testing.T) {
	const itemCount = 100
	require := require.New(t)
	objectsBefore := safepool.GetObjectsInUse()
	var cleanupCalls atomic.Int32
	owners, items := newConcurrentOwnershipPools(&cleanupCalls)
	owner := owners.Get()

	start := make(chan struct{})
	results := make(chan any, itemCount)
	var wg sync.WaitGroup
	for range itemCount {
		wg.Go(func() {
			<-start
			func() {
				defer func() { results <- recover() }()
				items.GetOwned(owner)
			}()
		})
	}
	close(start)
	wg.Wait()
	close(results)

	for result := range results {
		require.Nil(result)
	}
	require.Equal(
		objectsBefore+itemCount+1,
		safepool.GetObjectsInUse(),
	)

	owner.Release()
	require.Equal(int32(itemCount), cleanupCalls.Load())
	require.Equal(objectsBefore, safepool.GetObjectsInUse())
}

func TestGetOwnedConcurrentWithOwnerRelease(t *testing.T) {
	const attemptCount = 100
	require := require.New(t)
	objectsBefore := safepool.GetObjectsInUse()
	var cleanupCalls atomic.Int32
	owners, items := newConcurrentOwnershipPools(&cleanupCalls)
	owner := owners.Get()

	start := make(chan struct{})
	results := make(chan any, attemptCount)
	var wg sync.WaitGroup
	for range attemptCount {
		wg.Go(func() {
			<-start
			func() {
				defer func() { results <- recover() }()
				items.GetOwned(owner)
			}()
		})
	}
	releaseDone := make(chan any, 1)
	go func() {
		<-start
		defer func() { releaseDone <- recover() }()
		owner.Release()
	}()

	close(start)
	wg.Wait()
	close(results)
	require.Nil(<-releaseDone)

	succeeded := int32(0)
	for result := range results {
		if result != nil {
			require.Equal("owner already released", result)
		} else {
			succeeded++
		}
	}
	require.Equal(succeeded, cleanupCalls.Load())
	require.Equal(objectsBefore, safepool.GetObjectsInUse())
}

func TestGetOwnedRejectsReleasedOwner(t *testing.T) {
	require := require.New(t)
	objectsBefore := safepool.GetObjectsInUse()
	var cleanupCalls atomic.Int32
	owners, items := newConcurrentOwnershipPools(&cleanupCalls)
	owner := owners.Get()
	owner.Release()

	require.PanicsWithValue(
		"owner already released",
		func() { items.GetOwned(owner) },
	)
	require.Zero(cleanupCalls.Load())
	require.Equal(objectsBefore, safepool.GetObjectsInUse())
}
