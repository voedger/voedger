/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer_test

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	requirePkg "github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
)

func TestISequencer_Start(t *testing.T) {
	require := requirePkg.New(t)
	iTime := coreutils.MockTime
	storage := createDefaultStorage()
	storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
		isequencer.PLogOffset(0): {
			{
				Key:   isequencer.NumberKey{WSID: 1, SeqID: 1},
				Value: 100,
			},
		},
	})
	params := createDefaultParams(storage)

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

		// <-actualizeStartCh
		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.Equal(isequencer.PLogOffset(1), offset)

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

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		params := createDefaultParams(storage)
		params.MaxNumUnflushedValues = 5
		seq, cancel := isequencer.New(params, iTime)
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

		storage := createDefaultStorage()
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

		params := createDefaultParams(storage)
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1, 2: 1},
		}
		seq, cancel := isequencer.New(params, iTime)
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

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})
		params := createDefaultParams(storage)
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
		sequencer, cancel := isequencer.New(params, iTime)
		defer cancel()

		// Should panic when flush is called without an active transaction
		require.Panics(func() {
			sequencer.Flush()
		}, "Flush should panic when called without starting a transaction")
	})

	t.Run("should persist values to storage after flush completes", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.NewITime()

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		params := createDefaultParams(storage)
		sequencer, cancel := isequencer.New(params, iTime)
		defer cancel()

		firstOffset, err := storage.ReadNextPLogOffset()
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

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		sequencer, cancel := isequencer.New(createDefaultParams(storage), iTime)
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

		storage := createDefaultStorage()
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
		params := createDefaultParams(storage)
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: initialValue},
		}
		sequencer, cancel := isequencer.New(params, iTime)
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

		storage := createDefaultStorage()
		sequencer, cancel := isequencer.New(createDefaultParams(storage), iTime)
		defer cancel()

		require.Panics(func() {
			sequencer.Next(1)
		}, "Next should panic when called without starting a transaction")
	})

	t.Run("should panic for unknown sequence ID", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		sequencer, cancel := isequencer.New(createDefaultParams(storage), iTime)
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

		storage := createDefaultStorage()
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

		params := createDefaultParams(storage)
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {
				1: 1,
				2: 1,
			},
		}
		sequencer, cancel := isequencer.New(params, iTime)
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

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 300},
			},
		})

		seq, cancel := isequencer.New(createDefaultParams(storage), iTime)
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

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 400},
			},
		})

		// Use a tiny LRU cache that will definitely cause evictions
		params := createDefaultParams(storage)
		params.LRUCacheSize = 1 // Tiny cache to force evictions
		seq, cancel := isequencer.New(params, iTime)
		defer cancel()

		// First transaction
		isequencer.WaitForStart(t, seq, 1, 1, true)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(401), num)

		seq.Flush()

		// Second transaction - should still work even if LRU cache evicted the entry
		isequencer.WaitForStart(t, seq, 1, 1, true)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(402), num, "Next should handle LRU cache eviction correctly")

		seq.Flush()
	})

	t.Run("should use cached value after actualization", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 250},
			},
		})

		seq, cleanup := isequencer.New(createDefaultParams(storage), iTime)
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

		// Wait for actualization to complete
		isequencer.WaitForStart(t, seq, 1, 1, true)

		// This should now get value 251 (250+1) after actualization
		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(251), num, "Next should use latest value after actualization")

		seq.Flush()
	})
}

func TestISequencer_Actualize(t *testing.T) {
	require := requirePkg.New(t)
	mockedTime := coreutils.MockTime

	t.Run("not started -> panic", func(t *testing.T) {
		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{
					Key:   isequencer.NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		params := createDefaultParams(storage)
		seq, cleanup := isequencer.New(params, mockedTime)
		defer cleanup()
		require.Panics(func() { seq.Actualize() })
	})

	t.Run("empty plog", func(t *testing.T) {
		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{})
		params := createDefaultParams(storage)
		seq, cleanup := isequencer.New(params, mockedTime)
		defer cleanup()
		initialOffset := isequencer.WaitForStart(t, seq, 1, 1, true)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(2), num)

		// Actualize with empty PLog
		seq.Actualize()

		// Should be able to start a new transaction
		nextOffset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(initialOffset, nextOffset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(2), num)

		num2, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(3), num2)
	})

	t.Run("filled plog", func(t *testing.T) {
		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(0): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 2}, Value: 200},
			},
		})

		params := createDefaultParams(storage)
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1, 2: 1},
		}

		seq, cleanup := isequencer.New(params, mockedTime)
		defer cleanup()
		initialOffset := isequencer.WaitForStart(t, seq, 1, 1, true)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num)

		// Actualize with empty PLog
		seq.Actualize()

		// Should be able to start a new transaction
		nextOffset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(initialOffset, nextOffset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(101), num)

		num2, err := seq.Next(2)
		require.NoError(err)
		require.Equal(isequencer.Number(201), num2)
	})

}

// [~test.isequencer.MultipleActualizes~]
func TestISequencer_MultipleActualizes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := requirePkg.New(t)
	iTime := coreutils.NewITime()

	initialNumber := isequencer.Number(100)
	initialOffset := isequencer.PLogOffset(0)
	// Set up storage with initial values
	storage := isequencer.NewMockStorage(0, 0)
	storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
		initialOffset: {
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: initialNumber},
		},
	})

	params := createDefaultParams(storage)
	params.MaxNumUnflushedValues = 10
	params.LRUCacheSize = 100

	seq, cleanup := isequencer.New(params, iTime)
	defer cleanup()

	var currentOffset isequencer.PLogOffset
	prevOffset := initialOffset
	prevNumber := initialNumber
	countOfFlushes := 0
	// Run 100 cycles of operations
	const cycles = 100
	for i := 0; i < cycles; i++ {
		// Start transaction
		currentOffset = isequencer.WaitForStart(t, seq, 1, 1, true)
		currentNumber, err := seq.Next(1)
		require.NoError(err, "Failed to get next value in cycle %d", i)

		// Randomly choose between Flush and Actualize (with equal probability)
		if rand.Int()%2 == 0 {
			seq.Flush()
			// Simulate some time passing
			iTime.Sleep(50 * time.Millisecond)
			// Check if the offset and sequence number are incremented correctly
			require.Equal(currentOffset, prevOffset+1, "PLog offset should be incremented by 1 after flush")
			require.Equal(currentNumber, prevNumber+1, "Sequence number should be incremented by 1 after flush")
			// Update previous values
			prevOffset = currentOffset
			prevNumber = currentNumber
			countOfFlushes++
		} else {
			seq.Actualize()
		}

	}
	// Wait for all transactions to complete
	isequencer.WaitForStart(t, seq, 1, 1, true)
	// Check if the last offset in storage is equal to initial offset + count of flushes
	numbers, err := storage.ReadNumbers(1, []isequencer.SeqID{1})
	require.NoError(err)
	require.Equal(initialNumber+isequencer.Number(countOfFlushes), numbers[0], "Final number should be equal to initial number + count of flushes")
}

// [~test.isequencer.FlushPermanentlyFails~]
func TestISequencer_FlushPermanentlyFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := requirePkg.New(t)
	iTime := coreutils.NewITime()

	initialNumber := isequencer.Number(100)
	initialOffset := isequencer.PLogOffset(0)
	// Set up storage with initial values
	storage := isequencer.NewMockStorage(0, 0)
	storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
		initialOffset: {
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: initialNumber},
		},
	})

	params := createDefaultParams(storage)
	params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
		1: {1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1},
	}
	params.MaxNumUnflushedValues = 5

	seq, cleanup := isequencer.New(params, iTime)
	defer cleanup()
	// Set up retry count for infinite retry
	previousRetryCount := isequencer.RetryCount
	isequencer.RetryCount = 0
	// Set up storage to simulate a permanent failure
	storage.WriteValuesAndOffsetError = errors.New("some error")

	// first 6 sequence transaction will be ok
	for seqID := 1; seqID <= 6; seqID++ {
		isequencer.WaitForStart(t, seq, 1, 1, true)
		_, err := seq.Next(isequencer.SeqID(seqID))
		require.NoError(err)

		seq.Flush()
		// Simulate some time passing
		iTime.Sleep(50 * time.Millisecond)
	}
	// The 7th transaction should fail
	isequencer.WaitForStart(t, seq, 1, 1, false)
	// Turn off the error simulation for correct cleanup
	storage.WriteValuesAndOffsetError = nil
	// Reset the retry count to its original value
	isequencer.RetryCount = previousRetryCount
}

// [~test.isequencer.LongRecovery~]
func TestISequencer_LongRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := requirePkg.New(t)
	iTime := coreutils.NewITime()

	// Set up storage plog entries from 0 to 50
	// Generated by AI
	storage := isequencer.NewMockStorage(0, 0)
	storage.AddPLogEntry(0, 1, 1, 0)
	storage.AddPLogEntry(1, 1, 1, 1)
	storage.AddPLogEntry(2, 1, 1, 2)
	storage.AddPLogEntry(3, 1, 1, 3)
	storage.AddPLogEntry(4, 1, 1, 4)
	storage.AddPLogEntry(5, 1, 1, 5)
	storage.AddPLogEntry(6, 1, 1, 6)
	storage.AddPLogEntry(7, 1, 1, 7)
	storage.AddPLogEntry(8, 1, 1, 8)
	storage.AddPLogEntry(9, 1, 1, 9)
	storage.AddPLogEntry(10, 1, 1, 10)
	storage.AddPLogEntry(11, 1, 1, 11)
	storage.AddPLogEntry(12, 1, 1, 12)
	storage.AddPLogEntry(13, 1, 1, 13)
	storage.AddPLogEntry(14, 1, 1, 14)
	storage.AddPLogEntry(15, 1, 1, 15)
	storage.AddPLogEntry(16, 1, 1, 16)
	storage.AddPLogEntry(17, 1, 1, 17)
	storage.AddPLogEntry(18, 1, 1, 18)
	storage.AddPLogEntry(19, 1, 1, 19)
	storage.AddPLogEntry(20, 1, 1, 20)
	storage.AddPLogEntry(21, 1, 1, 21)
	storage.AddPLogEntry(22, 1, 1, 22)
	storage.AddPLogEntry(23, 1, 1, 23)
	storage.AddPLogEntry(24, 1, 1, 24)
	storage.AddPLogEntry(25, 1, 1, 25)
	storage.AddPLogEntry(26, 1, 1, 26)
	storage.AddPLogEntry(27, 1, 1, 27)
	storage.AddPLogEntry(28, 1, 1, 28)
	storage.AddPLogEntry(29, 1, 1, 29)
	storage.AddPLogEntry(30, 1, 1, 30)
	storage.AddPLogEntry(31, 1, 1, 31)
	storage.AddPLogEntry(32, 1, 1, 32)
	storage.AddPLogEntry(33, 1, 1, 33)
	storage.AddPLogEntry(34, 1, 1, 34)
	storage.AddPLogEntry(35, 1, 1, 35)
	storage.AddPLogEntry(36, 1, 1, 36)
	storage.AddPLogEntry(37, 1, 1, 37)
	storage.AddPLogEntry(38, 1, 1, 38)
	storage.AddPLogEntry(39, 1, 1, 39)
	storage.AddPLogEntry(40, 1, 1, 40)
	storage.AddPLogEntry(41, 1, 1, 41)
	storage.AddPLogEntry(42, 1, 1, 42)
	storage.AddPLogEntry(43, 1, 1, 43)
	storage.AddPLogEntry(44, 1, 1, 44)
	storage.AddPLogEntry(45, 1, 1, 45)
	storage.AddPLogEntry(46, 1, 1, 46)
	storage.AddPLogEntry(47, 1, 1, 47)
	storage.AddPLogEntry(48, 1, 1, 48)
	storage.AddPLogEntry(49, 1, 1, 49)
	storage.AddPLogEntry(50, 1, 1, 50)

	params := createDefaultParams(storage)
	params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
		1: {1: 1},
	}
	params.MaxNumUnflushedValues = 5
	// Simulate a long recovery process gradually from 0 to 50 offset
	for i := 0; i < 50; i++ {
		seq, cleanup := isequencer.New(params, iTime)
		storage.NextOffset = isequencer.PLogOffset(i)

		isequencer.WaitForStart(t, seq, 1, 1, true)

		num, err := seq.Next(isequencer.SeqID(1))
		require.NoError(err)

		require.Equal(isequencer.Number(51), num)

		cleanup()
	}
}

// createDefaultStorage creates a storage with default configuration and common test data
func createDefaultStorage() *isequencer.MockStorage {
	storage := isequencer.NewMockStorage(0, 0)

	return storage
}

// createDefaultParams creates default parameters for sequencer tests
func createDefaultParams(storage *isequencer.MockStorage) *isequencer.Params {
	return &isequencer.Params{
		SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1},
		},
		SeqStorage:            storage,
		MaxNumUnflushedValues: 5,
		LRUCacheSize:          1000,
		BatcherDelay:          5 * time.Millisecond,
	}
}
