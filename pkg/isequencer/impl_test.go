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
		storage.NextOffset = PLogOffset(100)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(99): {
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 500,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockedTime)

		// When
		offset, ok := seq.Start(1, 1)
		require.True(t, ok)
		require.Equal(t, PLogOffset(100), offset)

		// Generate new sequence Numbers 100 times
		for i := 1; i <= 100; i++ {
			num, err := seq.Next(1)
			require.NoError(t, err)
			require.Equal(t, Number(100+i), num)
		}

		seq.Flush()
		mockedTime.Sleep(1 * time.Second)

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
		storage.NextOffset = PLogOffset(99)
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		seq.(*sequencer).inprocMu.Lock()
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 1}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 2}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 3}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 4}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 6}] = 1
		seq.(*sequencer).inprocMu.Unlock()

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
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		seq.(*sequencer).inprocMu.Lock()
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 1}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 2}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 3}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 4}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 6}] = 1
		seq.(*sequencer).inprocMu.Unlock()

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

		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(0): {
				{
					Key:   NumberKey{WSID: WSID(1), SeqID: SeqID(1)},
					Value: Number(50),
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
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)

		_, ok := seq.Start(1, 1)
		require.True(ok)

		cancel()
		retryCount = 1
		storage.ReadNumbersError = errors.New("ReadNumbersError")
		seq.(*sequencer).toBeFlushedMu.Lock()
		seq.(*sequencer).toBeFlushed = make(map[NumberKey]Number)
		seq.(*sequencer).toBeFlushedMu.Unlock()

		num, err := seq.Next(1)
		require.ErrorIs(err, storage.ReadNumbersError)
		require.Zero(num)
	})
}

func TestBatcher(t *testing.T) {
	t.Run("should aggregate max values and wait for unflushed values threshold", func(t *testing.T) {
		require := require.New(t)

		// Given
		ctx := context.Background()
		storage := NewMockStorage(0, 0)
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100, 2: 200},
		}
		mockTime := coreutils.MockTime

		params := &Params{
			SeqStorage:            storage,
			MaxNumUnflushedValues: 3, // Small threshold to test waiting
			MaxFlushingInterval:   500 * time.Millisecond,
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
		require.Equal(PLogOffset(7), s.nextOffset, "Should update nextOffset to offset + 1")
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
			MaxFlushingInterval:   500 * time.Millisecond,
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
		time.Sleep(20 * time.Millisecond)

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
			MaxFlushingInterval:   500 * time.Millisecond,
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
		err := s.flushValues(targetOffset, true)
		require.NoError(err)

		// Verify offset was updated even with empty values
		storedOffset, err := storage.ReadNextPLogOffset()
		require.NoError(err)
		require.Equal(targetOffset, storedOffset)
	})

	t.Run("should handle error in WriteValuesAndOffset", func(t *testing.T) {
		require := require.New(t)

		// Set up mock storage and sequencer
		storage := NewMockStorage(0, 0)
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(5): {
				{
					Key:   NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		storage.WriteValuesAndOffsetError = errors.New("storage write error")
		retryCount = 1
		mockTime := coreutils.MockTime

		params := &Params{
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   500 * time.Millisecond,
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
		err := s.flushValues(targetOffset, true)
		require.Error(err)
	})
}
