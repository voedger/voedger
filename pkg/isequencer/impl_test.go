/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
)

func TestSequencer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Run("basic flow", func(t *testing.T) {
		mockedTime := coreutils.MockTime
		// Given
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(1): {
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 500,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockedTime)

		// When
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(t, PLogOffset(2), offset)

		// Generate new sequence Numbers 100 times
		for i := 1; i <= 100; i++ {
			num, err := seq.Next(1)
			require.NoError(t, err)
			require.Equal(t, Number(100+i), num)
		}

		seq.Flush()

		cleanup()
		seq.(*sequencer).flusherWG.Wait()

		nums, err := storage.ReadNumbers(1, []SeqID{1})
		require.NoError(t, err)
		require.Equal(t, Number(200), nums[0])
	})
}

func TestSequencer_Start(t *testing.T) {
	t.Run("should reject when too many unflushed values", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := NewMockStorage(0, 0)
		storage.NextOffset = PLogOffset(4)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): nil,
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		seq.(*sequencer).toBeFlushedMu.Lock()
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 2}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 3}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 4}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 6}] = 1
		seq.(*sequencer).toBeFlushedMu.Unlock()

		offset, ok := seq.Start(1, 1)
		require.False(t, ok)
		require.Zero(t, offset)
	})
}

func TestSequencer_Flush(t *testing.T) {
	t.Run("should reduce unflushed values and allow new transactions", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := NewMockStorage(0, 0)
		storage.NextOffset = PLogOffset(99)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): {
				{
					Key:   NumberKey{WSID: WSID(1), SeqID: SeqID(1)},
					Value: Number(100),
				},
			},
		})

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		seq.(*sequencer).toBeFlushedMu.Lock()
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 2}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 3}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 4}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 6}] = 1
		seq.(*sequencer).toBeFlushedMu.Unlock()

		offset, ok := seq.Start(1, 1)
		require.False(t, ok)
		require.Zero(t, offset)

		// Change the limit to allow new transactions
		seq.(*sequencer).params.MaxNumUnflushedValues = 100
		// Advance time to allow flushing to complete
		iTime.Add(time.Second)

		// Third transaction - should work after flush completes
		offset3, ok := seq.Start(1, 1)
		require.True(t, ok)
		require.NotZero(t, offset3)

		// Should be able to get the next sequence number after the previous ones
		num, err := seq.Next(1)
		_ = num
		require.NoError(t, err)

		seq.Flush()
	})
}

func TestSequencer_Next(t *testing.T) {
	t.Run("should return 0 when ReadNumbers() fails", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		initialNumber := Number(50)
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): {
				{
					Key:   NumberKey{WSID: WSID(1), SeqID: SeqID(1)},
					Value: initialNumber,
				},
			},
		})
		// No predefined sequence Numbers in storage
		initialValue := Number(1)
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: initialValue},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}, iTime)

		WaitForStart(t, seq, 1, 1, true)

		cancel()
		// Simulate a failure in ReadNumbers
		RetryCountMu.Lock()
		prevRetryCount := RetryCount
		RetryCount = 1
		RetryCountMu.Unlock()

		defer func() {
			RetryCountMu.Lock()
			RetryCount = prevRetryCount
			RetryCountMu.Unlock()
		}()

		storage.SetReadNumbersError(errors.New("ReadNumbersError"))
		seq.(*sequencer).toBeFlushedMu.Lock()
		seq.(*sequencer).toBeFlushed = make(map[NumberKey]Number)
		seq.(*sequencer).toBeFlushedMu.Unlock()

		num, err := seq.Next(1)
		require.ErrorIs(err, storage.ReadNumbersError)
		require.Zero(num)

	})

	t.Run("should update lru ", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		initialNumber := Number(50)
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(1): {
				{
					Key:   NumberKey{WSID: WSID(1), SeqID: SeqID(1)},
					Value: initialNumber,
				},
			},
		})
		// No predefined sequence Numbers in storage
		initialValue := Number(1)
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: initialValue},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		_, ok := seq.Start(1, 1)
		require.True(ok)

		num, err := seq.Next(1)
		require.ErrorIs(err, storage.ReadNumbersError)
		require.Equal(initialNumber+1, num)

		num, ok = seq.(*sequencer).lru.Get(NumberKey{WSID: 1, SeqID: 1})
		require.True(ok)
		require.Equal(initialNumber+1, num)
	})
}

func TestBatcher(t *testing.T) {
	t.Run("should aggregate max values and wait for unflushed values threshold", func(t *testing.T) {
		require := require.New(t)

		// Given
		ctx := context.Background()
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: Number(100),
				},
				{
					Key:   NumberKey{WSID: 1, SeqID: 2},
					Value: Number(200),
				},
			},
		})
		mockTime := coreutils.MockTime

		params := &Params{
			SeqStorage:            storage,
			MaxNumUnflushedValues: 3, // Small threshold to test waiting
			LRUCacheSize:          1000,
			BatcherDelay:          10 * time.Millisecond,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()
		s := seq.(*sequencer)

		// First, fill toBeFlushed to reach the threshold
		s.toBeFlushedMu.Lock()
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 101
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 2}] = 201
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 3}] = 301
		s.toBeFlushedOffset = 5
		s.toBeFlushedMu.Unlock()

		// Set up the batch to be processed
		batch := []SeqValue{
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102}, // Higher than existing
			{Key: NumberKey{WSID: 1, SeqID: 2}, Value: 201}, // Same as existing
			{Key: NumberKey{WSID: 1, SeqID: 4}, Value: 401}, // New key
		}

		// Launch batcher in a goroutine
		var err error
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = s.batcher(ctx, batch, 6)
		}()

		// Simulate flusher reducing the number of unflushed values
		s.toBeFlushedMu.Lock()
		delete(s.toBeFlushed, NumberKey{WSID: 1, SeqID: 2})
		s.toBeFlushedMu.Unlock()

		// Advance time for batcher delay
		mockTime.Add(15 * time.Millisecond)

		// Now batcher should proceed
		wg.Wait()
		// Verify no error
		require.NoError(err)

		// Verify toBeFlushed has been updated with maximum values
		s.toBeFlushedMu.RLock()
		defer s.toBeFlushedMu.RUnlock()

		// Should now have values from batch with max values preserved
		require.Equal(Number(102), s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}], "Should update with higher value")
		require.Equal(Number(301), s.toBeFlushed[NumberKey{WSID: 1, SeqID: 3}], "Should preserve existing higher value")
		require.Equal(Number(401), s.toBeFlushed[NumberKey{WSID: 1, SeqID: 4}], "Should add new value")

		// Verify offset was updated
		require.Equal(PLogOffset(7), s.toBeFlushedOffset, "Should update toBeFlushedOffset to offset + 1")

		// Verify nextOffset was updated
		s.nextOffsetMu.RLock()
		require.Equal(PLogOffset(7), s.nextOffset, "Should update nextOffset to offset + 1")
		s.nextOffsetMu.RUnlock()
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		require := require.New(t)

		// Given
		ctx, cancel := context.WithCancel(context.Background())
		storage := NewMockStorage(0, 0)
		mockTime := coreutils.MockTime

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 1, // Small threshold to force waiting
			LRUCacheSize:          1000,
			BatcherDelay:          100 * time.Millisecond,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()
		s := seq.(*sequencer)

		// Fill toBeFlushed to reach the threshold
		s.toBeFlushedMu.Lock()
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 101
		s.toBeFlushedOffset = 5
		s.toBeFlushedMu.Unlock()

		// Set up the batch to be processed
		batch := []SeqValue{
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102},
		}

		// Launch batcher in a goroutine
		var err error
		done := make(chan struct{})
		go func() {
			err = s.batcher(ctx, batch, 6)
			close(done)
		}()

		// Wait briefly to ensure batcher is waiting
		// time.Sleep(20 * time.Millisecond)

		// Cancel the context
		cancel()

		// Batcher should exit with context error
		select {
		case <-done:
			// Expected
		case <-time.After(500 * time.Millisecond):
			require.Fail("Batcher should have exited after context cancellation")
		}

		// Verify context error
		require.Error(err)
		require.Equal(context.Canceled, err)
	})
}

func TestSequencer_FlushValues(t *testing.T) {
	t.Run("should handle empty toBeFlushed map", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): nil,
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		mockTime := coreutils.MockTime

		params := &Params{
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()
		s := seq.(*sequencer)

		// Set empty toBeFlushed
		s.toBeFlushedMu.Lock()
		s.toBeFlushed = map[NumberKey]Number{}
		targetOffset := PLogOffset(7)
		s.toBeFlushedOffset = targetOffset
		s.toBeFlushedMu.Unlock()

		// Test with empty values
		err := s.flushValues(targetOffset)
		require.NoError(err)

		// Verify offset was updated even with empty values
		storedOffset, err := storage.ReadNextPLogOffset()
		require.NoError(err)
		require.Equal(targetOffset, storedOffset)
	})

	t.Run("should handle error in WriteValuesAndNextPLogOffset", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		storage.SetWriteValuesAndOffset(errors.New("storage write error"))

		RetryCountMu.Lock()
		prevRetryCount := RetryCount
		RetryCount = 1
		RetryCountMu.Unlock()

		defer func() {
			RetryCountMu.Lock()
			RetryCount = prevRetryCount
			RetryCountMu.Unlock()
		}()
		mockTime := coreutils.MockTime

		params := &Params{
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockTime)
		s := seq.(*sequencer)

		// Set empty toBeFlushed
		s.toBeFlushedMu.Lock()
		s.toBeFlushed = map[NumberKey]Number{}
		targetOffset := PLogOffset(7)
		s.toBeFlushedOffset = targetOffset
		s.toBeFlushedMu.Unlock()

		cleanup()
		// Test with empty values
		err := s.flushValues(targetOffset)
		require.Error(err)
	})
}

func TestNextNumberSourceOrder(t *testing.T) {
	t.Run("check the value is cached after next", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): nil,
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		mockTime := coreutils.MockTime

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()

		numberKey := NumberKey{WSID: 1, SeqID: 1}

		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)
		numInitial, err := seq.Next(1)
		require.NoError(err)
		require.NotZero(numInitial)
		numCached, ok := seq.(*sequencer).lru.Get(numberKey)
		require.True(ok)
		require.Equal(numInitial, numCached)

		seq.Actualize()
	})

	t.Run("check taken from lru on next", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): nil,
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		mockTime := coreutils.MockTime

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()

		numberKey := NumberKey{WSID: 1, SeqID: 1}

		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)

		// tamper the cache to ensure we'll use cache on Next
		seq.(*sequencer).lru.Add(numberKey, 10001)

		// expect read from cache first on normal Next call
		numFromCache, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(10002), numFromCache)

		seq.Actualize()
	})

	t.Run("missing in lru -> take from inproc", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): nil,
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		mockTime := coreutils.MockTime

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()

		numberKey := NumberKey{WSID: 1, SeqID: 1}

		// start
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)

		// fill the cache
		numInitial, err := seq.Next(1)
		require.NoError(err)
		require.NotZero(numInitial)

		// evict the cached number
		require.True(seq.(*sequencer).lru.Remove(numberKey))

		// tamper inproc to be sure we'll read exactly from inproc in this case
		seq.(*sequencer).inproc[numberKey] = 20001

		// missing in cache -> expect read from inproc
		numActual, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(20002), numActual)

		seq.Actualize()
	})

	t.Run("missing in lru and in inproc -> take from toBeFlushed", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): nil,
			PLogOffset(1): nil,
			PLogOffset(2): nil,
			PLogOffset(3): nil,
			PLogOffset(4): nil,
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		mockTime := coreutils.MockTime

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockTime)
		defer cleanup()

		numberKey := NumberKey{WSID: 1, SeqID: 1}

		// start
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)

		// fill the cache and inproc
		numInitial, err := seq.Next(1)
		require.NoError(err)
		require.NotZero(numInitial)

		// clear inproc + keep toBeFlushed filled by making flusher() stuck
		continueCh := make(chan any)
		storage.SetOnWriteValuesAndOffset(func() {
			<-continueCh
		})
		defer func() {
			storage.SetOnWriteValuesAndOffset(nil)
		}()
		seq.Flush()
		seq.(*sequencer).inproc = map[NumberKey]Number{}

		// clear cache
		removed := seq.(*sequencer).lru.Remove(numberKey)
		require.True(removed)

		// tamper toBeFlushed to ensure we'll read exactly from toBeFlushed in this case
		seq.(*sequencer).toBeFlushedMu.Lock()
		seq.(*sequencer).toBeFlushed[numberKey] = 30001
		seq.(*sequencer).toBeFlushedMu.Unlock()

		offset = WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(7), offset)

		// missing in cache and in inproc -> expect read from toBeFlushed
		numFromToBeFlushed, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(30002), numFromToBeFlushed)
		close(continueCh)

		seq.Actualize()
	})
}
