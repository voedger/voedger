/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"errors"
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
		storage := NewMockStorage()
		storage.nextOffset = PLogOffset(99)
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}
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
		require.Equal(t, nums[0], Number(200))
	})

	t.Run("actualization", func(t *testing.T) {
		mockedTime := coreutils.MockTime
		// Given
		storage := NewMockStorage()
		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 500,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, mockedTime)
		defer cleanup()
		seq.(*sequencer).actualizerWG.Wait()

		// When
		_, ok := seq.Start(1, 1)
		require.True(t, ok)

		_, err := seq.Next(1)
		require.NoError(t, err)

		seq.Actualize()

		// Then
		_, ok = seq.Start(1, 1)
		require.False(t, ok, "should not start during actualization")

		// FIXME: check further New, Start, Next will return 1 again after Actuazlier
	})
}

func TestSequencer_Start(t *testing.T) {
	t.Run("should reject when too many unflushed values", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.nextOffset = PLogOffset(99)
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

	t.Run("should start successfully after flush completes", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := require.New(t)

		storage := NewMockStorage()
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// First transaction
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.Equal(PLogOffset(1), offset)

		count := 3
		// Generate sequence Numbers
		for i := 0; i < count; i++ {
			num, err := seq.Next(1)
			require.NoError(err)
			require.Equal(Number(100+i+1), num)
		}

		seq.Flush()

		// Start a new transaction
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.Equal(PLogOffset(2), offset)

		// Verify we can get the next number in sequence
		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(100+count+1), num, "Sequence should continue from last value")
	})
}

func TestBatcher(t *testing.T) {
	t.Run("should aggregate max values and write to storage", func(t *testing.T) {
		// Given
		storage := NewMockStorage()
		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 100, 2: 200},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 0,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, coreutils.MockTime)
		defer cleanup()

		// When
		batch := []SeqValue{
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 101},
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102}, // Higher value for same key
			{Key: NumberKey{WSID: 1, SeqID: 2}, Value: 201},
			{Key: NumberKey{WSID: 1, SeqID: 2}, Value: 200}, // Lower value for same key
		}
		batchOffset := PLogOffset(10)

		err := seq.(*sequencer).batcher(batch, batchOffset)
		require.NoError(t, err)

		// Then
		// Verify storage received max values
		nums, err := storage.ReadNumbers(1, []SeqID{1, 2})
		require.NoError(t, err)
		require.Equal(t, []Number{102, 201}, nums)

		// Verify offset was written
		offset, err := storage.ReadNextPLogOffset()
		require.NoError(t, err)
		require.Equal(t, PLogOffset(11), offset)
	})

	t.Run("should handle empty batch", func(t *testing.T) {
		// Given
		storage := NewMockStorage()
		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 0,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}

		seq, cleanup := New(params, coreutils.MockTime)
		defer cleanup()

		// When
		err := seq.(*sequencer).batcher([]SeqValue{}, PLogOffset(1))
		require.NoError(t, err)

		// Then
		offset, err := storage.ReadNextPLogOffset()
		require.NoError(t, err)
		require.Equal(t, PLogOffset(2), offset)
	})

	t.Run("should handle storage write errors", func(t *testing.T) {
		//t.Skip()
		// Given
		storage := NewMockStorage()
		params := &Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 0,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}

		retryCount = 2
		seq, cleanup := New(params, coreutils.NewITime())
		cleanup()

		batch := []SeqValue{
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 101},
		}

		storage.writeValuesAndOffsetError = errors.New("write error")
		// When
		err := seq.(*sequencer).batcher(batch, PLogOffset(1))

		// Then
		require.Error(t, err)
	})
}

func TestSequencer_Flush(t *testing.T) {
	t.Run("should reduce unflushed values and allow new transactions", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.nextOffset = PLogOffset(99)
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

func TestSequencer_Actualize(t *testing.T) {
	t.Run("should clear all cached data and process PLog", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}

		// Add entries to the mock PLog to simulate actualization
		storage.pLog = [][]SeqValue{
			{
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 200},
			},
		}

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Start transaction to set up state
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Get a number to populate cache
		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(201), num)

		// Start actualization
		seq.Actualize()

		// Wait for actualization to start
		for {
			if seq.(*sequencer).actualizerInProgress.Load() {
				break
			}
			time.Sleep(time.Millisecond)
		}

		// Try to start a new transaction - should be rejected during actualization
		offset, ok = seq.Start(1, 1)
		require.False(ok, "Start should be rejected while actualization is in progress")
		require.Zero(offset)

		// Wait for actualization to complete
		seq.(*sequencer).actualizerWG.Wait()

		// Now should be able to start a transaction
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Number should be updated from PLog
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(Number(201), num, "Next should return value incremented from actualized PLog value")

		seq.Flush()
	})

	t.Run("should work correctly when PLog is empty", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}

		// Empty PLog
		storage.pLog = [][]SeqValue{}

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 50},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(101), num)

		// Actualize with empty PLog
		seq.Actualize()
		seq.(*sequencer).actualizerWG.Wait()

		// Should be able to start a new transaction
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Value should remain unchanged since PLog is empty
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(Number(101), num, "Sequence should remain unchanged when PLog is empty")

		seq.Flush()
	})

	t.Run("should update sequence Numbers to match PLog after actualization", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100, 2: 200},
		}

		// Set up PLog with higher sequence values
		storage.pLog = [][]SeqValue{
			{
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 150},
				{Key: NumberKey{WSID: 1, SeqID: 2}, Value: 250},
			},
		}

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 50, 2: 150},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Start new transaction after actualization
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Actualize and wait
		seq.Actualize()
		seq.(*sequencer).actualizerWG.Wait()

		// Start new transaction after actualization
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Values should be updated from PLog
		num1, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(151), num1, "Sequence 1 should be updated from PLog")

		num2, err := seq.Next(2)
		require.NoError(err)
		require.Equal(Number(251), num2, "Sequence 2 should be updated from PLog")

		seq.Flush()
	})

	t.Run("should handle multiple actualizations", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}

		// First set of PLog entries
		storage.pLog = [][]SeqValue{
			{
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 150},
			},
		}

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 50},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// First transaction and actualization
		_, ok := seq.Start(1, 1)
		require.True(ok)
		seq.Actualize()
		seq.(*sequencer).actualizerWG.Wait()

		// Update PLog with new entries
		storage.pLog = [][]SeqValue{
			{
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 200},
			},
		}

		// Second transaction and actualization
		_, ok = seq.Start(1, 1)
		require.True(ok)
		seq.Actualize()
		seq.(*sequencer).actualizerWG.Wait()

		// Third transaction to verify sequence values
		_, ok = seq.Start(1, 1)
		require.True(ok)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(151), num, "Sequence should be updated after multiple actualizations")

		seq.Flush()
	})

	t.Run("cleaning up toBeFlushed and inproc after Actualize", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 100},
		}

		// First set of PLog entries
		storage.pLog = [][]SeqValue{
			{
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 150},
			},
		}

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 50},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Set up inproc and toBeFlushed
		seq.(*sequencer).inprocMu.Lock()
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 1}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 2}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 3}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 4}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).inproc[NumberKey{WSID: 1, SeqID: 6}] = 1
		seq.(*sequencer).inprocMu.Unlock()

		seq.(*sequencer).toBeFlushedMu.Lock()
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 2}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 3}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 4}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).toBeFlushed[NumberKey{WSID: 1, SeqID: 5}] = 1
		seq.(*sequencer).toBeFlushedMu.Unlock()
		// First transaction and actualization
		_, ok := seq.Start(1, 1)
		require.True(ok)
		seq.Actualize()
		seq.(*sequencer).actualizerWG.Wait()

		// Verify inproc and toBeFlushed are empty after Actualize
		require.Empty(seq.(*sequencer).inproc)
		require.Empty(seq.(*sequencer).toBeFlushed)
	})
}

func TestSequencer_Next(t *testing.T) {
	t.Run("should use cached value after actualization", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 200},
		}

		// Add entries to the mock PLog to simulate actualization
		storage.pLog = [][]SeqValue{
			{
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 250},
			},
		}

		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// First transaction
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// This should get value from storage (200) and increment to 201
		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(251), num)

		// Actualize to process the PLog entry (with value 250)
		seq.Actualize()
		seq.(*sequencer).actualizerWG.Wait() // Give time for actualization to complete

		// Start a new transaction
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// This should now get value 251 (250+1) after actualization
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(Number(251), num, "Next should use latest value after actualization")

		seq.Flush()
	})

	t.Run("should handle LRU cache eviction correctly", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 400},
		}

		// Use a tiny LRU cache that will definitely cause evictions
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1, // Tiny cache to force evictions
		}, iTime)
		defer cancel()

		// First transaction
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(401), num)

		seq.Flush()

		// Second transaction - should still work even if LRU cache evicted the entry
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(Number(402), num, "Next should handle LRU cache eviction correctly")

		seq.Flush()
	})

	t.Run("should maintain proper sequence across multiple transactions", func(t *testing.T) {
		require := require.New(t)
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		storage.Numbers = map[WSID]map[SeqID]Number{
			1: {1: 300},
		}
		seq, cancel := New(&Params{
			SeqTypes: map[WSKind]map[SeqID]Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Transaction 1
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(301), num, "Sequence should continue from last value")

		seq.Flush()

		// Transaction 2
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(Number(302), num, "Sequence should continue from last value")

		seq.Flush()
	})

}
