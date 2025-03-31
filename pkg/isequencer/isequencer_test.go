/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer_test

import (
	"context"
	"testing"
	"time"

	requirePkg "github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
)

var actualizationTimeoutLimit = 5 * time.Second

func TestISequencer_Start(t *testing.T) {
	require := requirePkg.New(t)
	iTime := coreutils.MockTime
	storage := isequencer.NewMockStorage(0, 0)
	params := &isequencer.Params{
		SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1},
		},
		SeqStorage:            storage,
		MaxNumUnflushedValues: 5,
		MaxFlushingInterval:   500 * time.Millisecond,
		LRUCacheSize:          1000,
	}

	t.Run("should panic when cleanup process is initiated", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, iTime)
		cleanup()
		require.Panics(func() {
			sequencer.Start(1, 1)
		})
	})

	t.Run("should panic when transaction already started", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, iTime)
		defer cleanup()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(t, offset)

		require.Panics(func() {
			sequencer.Start(1, 1)
		})
	})

	t.Run("should reject when actualization in progress", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, iTime)
		defer cleanup()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.Zero(offset)

		sequencer.Actualize()

		// Should be rejected while actualization is in progress
		offset, ok = sequencer.Start(1, 1)
		require.False(ok)
		require.Zero(offset)
	})

	t.Run("should panic when unknown wsKind", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, iTime)
		defer cleanup()

		require.Panics(func() {
			sequencer.Start(2, 1) // WSKind 2 is not defined
		})
	})

	t.Run("should start successfully after flush completes", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
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
		require.Equal(isequencer.PLogOffset(1), offset)

		count := 3
		// Generate sequence Numbers
		for i := 0; i < count; i++ {
			num, err := seq.Next(1)
			require.NoError(err)
			require.Equal(isequencer.Number(100+i+1), num)
		}

		seq.Flush()

		// Start a new transaction
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.Equal(isequencer.PLogOffset(2), offset)

		// Verify we can get the next number in sequence
		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(100+count+1), num, "Sequence should continue from last value")
	})
}

func TestISequencer_Flush(t *testing.T) {
	t.Run("should correctly increment sequence values after flush", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(2)},
					Value: isequencer.Number(200),
				},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1, 2: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Start transaction and get some values
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Get values for both sequence types
		num1, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num1)

		num2, err := seq.Next(2)
		require.NoError(err)
		require.Equal(isequencer.Number(201), num2)

		// Flush values
		seq.Flush()

		// Advance time to ensure flush completes
		iTime.Add(time.Second)

		// Start a new transaction
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Get next values - should be incremented from previous values
		nextNum1, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(102), nextNum1, "Sequence should continue from last value after flush")

		nextNum2, err := seq.Next(2)
		require.NoError(err)
		require.Equal(isequencer.Number(202), nextNum2, "Sequence should continue from last value after flush")

		seq.Flush()
	})

	t.Run("should panic when called without starting a transaction", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Should panic when flush is called without an active transaction
		require.Panics(func() {
			sequencer.Flush()
		}, "Flush should panic when called without starting a transaction")
	})

	t.Run("should persist values to storage after flush completes", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.NewITime()

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		firstOffset, err := storage.ReadNextPLogOffset()
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond, // Short interval to ensure flush happens quickly
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Start transaction and generate a value
		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num)

		// Flush and advance time
		sequencer.Flush()
		iTime.Sleep(100 * time.Millisecond) // Advance time beyond flush interval

		// Verify the value was written to storage
		numbers, err := storage.ReadNumbers(1, []isequencer.SeqID{1})
		require.NoError(err)
		require.Equal(isequencer.Number(101), numbers[0], "Flushed value should be persisted in storage")

		// Verify next PLog offset was updated
		nextOffset, err := storage.ReadNextPLogOffset()
		require.NoError(err)
		require.Equal(isequencer.PLogOffset(1), nextOffset-firstOffset)
	})
}

func TestISequencer_Next(t *testing.T) {
	t.Run("should return incremented sequence number", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		_, ok := sequencer.Start(1, 1)
		require.True(ok)

		num, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num, "Next should return incremented sequence number")

		// Call Next again for the same sequence - should increment again
		num2, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(102), num2, "Subsequent call to Next should increment further")

		sequencer.Flush()
	})

	t.Run("should use initial value when sequence number not in storage", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.NextOffset = 1
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(50),
				},
			},
		})
		// No predefined sequence Numbers in storage
		initialValue := isequencer.Number(1)
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: initialValue},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		_, ok := sequencer.Start(1, 1)
		require.True(ok)

		num, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(initialValue+1, num, "Next should use initial value when sequence not in storage")

		sequencer.Flush()
	})

	t.Run("should panic when called without starting transaction", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		require.Panics(func() {
			sequencer.Next(1)
		}, "Next should panic when called without starting a transaction")
	})

	t.Run("should panic for unknown sequence ID", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1}, // Only sequence ID 1 is defined
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		_, ok := sequencer.Start(1, 1)
		require.True(ok)

		require.Panics(func() {
			sequencer.Next(2) // Sequence ID 2 is not defined
		}, "Next should panic for unknown sequence ID")

		sequencer.Flush()
	})

	t.Run("should handle multiple sequence types correctly", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(2)},
					Value: isequencer.Number(200),
				},
			},
		})

		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {
					1: 1,
					2: 1,
				},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		_, ok := sequencer.Start(1, 1)
		require.True(ok)

		// Get next value for sequence 1
		num1, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num1)

		// Get next value for sequence 2
		num2, err := sequencer.Next(2)
		require.NoError(err)
		require.Equal(isequencer.Number(201), num2)

		// Get another value for sequence 1 - should increment
		num1Again, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(102), num1Again)

		sequencer.Flush()
	})

	t.Run("should maintain proper sequence across multiple transactions", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 300},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
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

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(301), num, "Sequence should continue from last value")

		seq.Flush()

		// Transaction 2
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(302), num, "Sequence should continue from last value")

		seq.Flush()
	})

	t.Run("should handle LRU cache eviction correctly", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 400},
			},
		})

		// Use a tiny LRU cache that will definitely cause evictions
		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
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
		require.Equal(isequencer.Number(401), num)

		seq.Flush()

		// Second transaction - should still work even if LRU cache evicted the entry
		offset, ok = seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(402), num, "Next should handle LRU cache eviction correctly")

		seq.Flush()
	})

	t.Run("should use cached value after actualization", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 250},
			},
		})

		seq, cleanup := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cleanup()

		// First transaction
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// This should get value from storage (200) and increment to 201
		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(251), num)

		// Actualize to process the PLog entry (with value 250)
		seq.Actualize()

		// Start a new transaction
		timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel()
		offset2, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset2 == 0 && timeoutCtx.Err() == nil {
			offset2, ok = seq.Start(1, 1)
		}

		// This should now get value 251 (250+1) after actualization
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(251), num, "Next should use latest value after actualization")

		seq.Flush()
	})
}

func TestISequencer_Actualize(t *testing.T) {
	t.Run("actualization basic flow", func(t *testing.T) {
		require := requirePkg.New(t)
		mockedTime := coreutils.MockTime
		// Given
		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		params := &isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 500,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}

		seq, cleanup := isequencer.New(params, mockedTime)

		// When
		offsetBegin, ok := seq.Start(1, 1)
		_ = offsetBegin
		require.True(ok)

		_, err := seq.Next(1)
		require.NoError(err)

		seq.Actualize()

		timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel()
		offset2, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset2 == 0 && timeoutCtx.Err() == nil {
			offset2, ok = seq.Start(1, 1)
		}
		require.True(ok)

		cleanup()

		seq2, cleanup2 := isequencer.New(params, mockedTime)
		defer cleanup2()

		offsetAfter, ok := seq2.Start(1, 1)
		_ = offsetAfter
		require.True(ok)

		_, err = seq2.Next(1)
		require.NoError(err)

		require.Equal(isequencer.PLogOffset(1), offsetAfter)
		require.Equal(offsetAfter, offsetBegin)
	})

	t.Run("should panic when called without starting transaction", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// Should panic when actualize is called without an active transaction
		require.Panics(func() {
			sequencer.Actualize()
		}, "Actualize should panic when called without starting a transaction")
	})

	t.Run("should handle multiple actualizations", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 150},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
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

		// Update PLog with new entries
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 150},
			},
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 200},
			},
		})

		// Second transaction and actualization
		timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel()
		offset2, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset2 == 0 && timeoutCtx.Err() == nil {
			offset2, ok = seq.Start(1, 1)
		}
		require.True(ok)

		seq.Actualize()

		// Third transaction to verify sequence values
		timeoutCtx2, timeoutCtxCancel2 := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel2()
		offset3, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset3 == 0 && timeoutCtx2.Err() == nil {
			offset3, ok = seq.Start(1, 1)
		}
		require.True(ok)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(201), num, "Sequence should be updated after multiple actualizations")

		seq.Flush()
	})

	t.Run("should update sequence Numbers to match PLog after actualization", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.Numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100, 2: 200},
		}

		// Set up PLog with higher sequence values
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 150},
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 2}, Value: 250},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
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

		// Start new transaction after actualization
		timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel()
		offset2, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset2 == 0 && timeoutCtx.Err() == nil {
			offset2, ok = seq.Start(1, 1)
		}
		require.True(ok)

		// Values should be updated from PLog
		num1, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(151), num1, "Sequence 1 should be updated from PLog")

		num2, err := seq.Next(2)
		require.NoError(err)
		require.Equal(isequencer.Number(251), num2, "Sequence 2 should be updated from PLog")

		seq.Flush()
	})

	t.Run("should clear all cached data and process PLog", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.Numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}

		// Add entries to the mock PLog to simulate actualization
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 200},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
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
		require.Equal(isequencer.Number(201), num)

		// Start actualization
		seq.Actualize()

		// Now should be able to start a transaction
		timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel()
		offset2, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset2 == 0 && timeoutCtx.Err() == nil {
			offset2, ok = seq.Start(1, 1)
		}
		require.True(ok)

		// Number should be updated from PLog
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(201), num, "Next should return value incremented from actualized PLog value")

		seq.Flush()
	})

	t.Run("should work correctly when PLog is empty", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage(0, 0)
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		seq, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 50},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		_, ok := seq.Start(1, 1)
		require.True(ok)
		// require.NotZero(offset)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num)

		// Actualize with empty PLog
		seq.Actualize()

		// Should be able to start a new transaction
		timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
		defer timeoutCtxCancel()
		offset2, ok := seq.Start(1, 1)
		// Wait for the actualization to complete
		for offset2 == 0 && timeoutCtx.Err() == nil {
			offset2, ok = seq.Start(1, 1)
		}
		require.True(ok)

		// Value should remain unchanged since PLog is empty
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num, "Sequence should remain unchanged when PLog is empty")

		seq.Flush()
	})
}
