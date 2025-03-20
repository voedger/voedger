/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer_test

import (
	"testing"
	"time"

	requirePkg "github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
)

func TestISequencer_Start(t *testing.T) {
	require := requirePkg.New(t)
	iTime := coreutils.MockTime
	storage := isequencer.NewMockStorage()
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
		require.NotZero(offset)

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
}

func TestISequencer_Flush(t *testing.T) {
	t.Run("should correctly increment sequence values after flush", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := isequencer.NewMockStorage()
		storage.Numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100, 2: 200},
		}
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
		// FIXME: no guarantee here that the flush is actually completed (better check storage)

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

		storage := isequencer.NewMockStorage()
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
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage()
		storage.Numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
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
		iTime.Add(50 * time.Millisecond) // Advance time beyond flush interval

		// Verify the value was written to storage
		numbers, err := storage.ReadNumbers(1, []isequencer.SeqID{1})
		require.NoError(err)
		require.Equal(isequencer.Number(100), numbers[0], "Flushed value should be persisted in storage")

		// Verify next PLog offset was updated
		nextOffset, err := storage.ReadNextPLogOffset()
		require.NoError(err)
		require.NotZero(nextOffset, "Next PLog offset should be updated after flush")
	})
}

func TestISequencer_Next(t *testing.T) {
	t.Run("should return incremented sequence number", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage()
		storage.Numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
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

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

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

		storage := isequencer.NewMockStorage()
		// No predefined sequence Numbers in storage
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 50}, // Initial value is 50
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		num, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(51), num, "Next should use initial value when sequence not in storage")

		sequencer.Flush()
	})

	t.Run("should panic when called without starting transaction", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage()
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

		storage := isequencer.NewMockStorage()
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

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		require.Panics(func() {
			sequencer.Next(2) // Sequence ID 2 is not defined
		}, "Next should panic for unknown sequence ID")

		sequencer.Flush()
	})
	t.Run("should handle multiple sequence types correctly", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage()
		storage.Numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {
				1: 100, // First sequence starts at 100
				2: 200, // Second sequence starts at 200
			},
		}
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {
					1: 1, // Initial value for sequence 1
					2: 2, // Initial value for sequence 2
				},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 10,
			MaxFlushingInterval:   10 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

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
}

func TestISequencer_Actualize(t *testing.T) {
	t.Run("should panic when called without starting transaction", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := isequencer.NewMockStorage()
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 100},
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
}
