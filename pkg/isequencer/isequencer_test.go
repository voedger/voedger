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
		isequencer.PLogOffset(1): {
			{
				Key:   isequencer.NumberKey{WSID: 1, SeqID: 1},
				Value: 100,
			},
		},
	})
	params := createDefaultParams()

	t.Run("should panic when cleanup process is initiated", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, storage, iTime)
		cleanup()
		require.Panics(func() {
			sequencer.Start(1, 1)
		})
	})

	t.Run("should panic when transaction already started", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, storage, iTime)
		defer cleanup()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(t, offset)

		require.Panics(func() {
			sequencer.Start(1, 1)
		})
	})

	t.Run("should reject when actualization in progress", func(t *testing.T) {
		seq, cleanup := isequencer.New(params, storage, iTime)
		defer cleanup()

		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(isequencer.PLogOffset(2), offset)

		seq.Actualize()

		// Should be rejected while actualization is in progress
		offset, ok := seq.Start(1, 1)
		require.False(ok)
		require.Zero(offset)
	})

	t.Run("should panic when unknown wsKind", func(t *testing.T) {
		sequencer, cleanup := isequencer.New(params, storage, iTime)
		defer cleanup()

		require.Panics(func() {
			sequencer.Start(2, 1) // WSKind 2 is not defined
		})
	})

	t.Run("should start successfully after flush completes", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		params := createDefaultParams()
		params.MaxNumUnflushedValues = 5
		seq, cancel := isequencer.New(params, storage, iTime)
		defer cancel()

		// First transaction
		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.Equal(isequencer.PLogOffset(2), offset)

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
		require.Equal(isequencer.PLogOffset(3), offset)

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
			isequencer.PLogOffset(1): {
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

		params := createDefaultParams()
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1, 2: 1},
		}
		seq, cancel := isequencer.New(params, storage, iTime)
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
			isequencer.PLogOffset(1): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})
		params := createDefaultParams()
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
		sequencer, cancel := isequencer.New(params, storage, iTime)
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
			isequencer.PLogOffset(1): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		params := createDefaultParams()
		sequencer, cancel := isequencer.New(params, storage, iTime)
		defer cancel()

		firstOffset, err := storage.ReadNextPLogOffset()
		require.NoError(err)
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
			isequencer.PLogOffset(1): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: isequencer.Number(100),
				},
			},
		})

		sequencer, cancel := isequencer.New(createDefaultParams(), storage, iTime)
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

		initialValue := isequencer.Number(50)
		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(1): {
				{
					Key:   isequencer.NumberKey{WSID: isequencer.WSID(1), SeqID: isequencer.SeqID(1)},
					Value: initialValue,
				},
			},
		})
		// No predefined sequence Numbers in storage
		params := createDefaultParams()
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1},
		}
		sequencer, cancel := isequencer.New(params, storage, iTime)
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
		sequencer, cancel := isequencer.New(createDefaultParams(), storage, iTime)
		defer cancel()

		require.Panics(func() {
			sequencer.Next(1)
		}, "Next should panic when called without starting a transaction")
	})

	t.Run("should panic for unknown sequence ID", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		sequencer, cancel := isequencer.New(createDefaultParams(), storage, iTime)
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
			isequencer.PLogOffset(1): {
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

		params := createDefaultParams()
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {
				1: 1,
				2: 1,
			},
		}
		sequencer, cancel := isequencer.New(params, storage, iTime)
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
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 300},
			},
		})

		seq, cancel := isequencer.New(createDefaultParams(), storage, iTime)
		defer cancel()

		// Transaction 1
		offset, ok := seq.Start(1, 1)
		require.NotZero(offset)
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

	t.Run("should handle cache eviction correctly", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 400},
			},
		})

		// Use a tiny LRU cache that will definitely cause evictions
		params := createDefaultParams()
		params.LRUCacheSize = 1 // Tiny cache to force evictions
		seq, cancel := isequencer.New(params, storage, iTime)
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
		require.Equal(isequencer.Number(402), num, "Next should handle cache eviction correctly")

		seq.Flush()
	})

	t.Run("should use cached value after actualization", func(t *testing.T) {
		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 250},
			},
		})

		seq, cleanup := isequencer.New(createDefaultParams(), storage, iTime)
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
			isequencer.PLogOffset(1): {
				{
					Key:   isequencer.NumberKey{WSID: 1, SeqID: 1},
					Value: 100,
				},
			},
		})
		params := createDefaultParams()
		seq, cleanup := isequencer.New(params, storage, mockedTime)
		defer cleanup()
		require.Panics(func() { seq.Actualize() })
	})

	t.Run("empty plog", func(t *testing.T) {
		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{})
		params := createDefaultParams()
		seq, cleanup := isequencer.New(params, storage, mockedTime)
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
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 2}, Value: 200},
			},
		})

		params := createDefaultParams()
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {1: 1, 2: 1},
		}

		seq, cleanup := isequencer.New(params, storage, mockedTime)
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

// [~server.design.sequences/test.isequencer.MultipleActualizes~impl]
func TestISequencer_MultipleActualizes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := requirePkg.New(t)
	iTime := coreutils.NewITime()

	initialNumber := isequencer.Number(100)
	initialOffset := isequencer.PLogOffset(1)
	// Set up storage with initial values
	storage := isequencer.NewMockStorage(0, 0)
	storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
		initialOffset: {
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: initialNumber},
		},
	})

	params := createDefaultParams()
	params.MaxNumUnflushedValues = 10
	params.LRUCacheSize = 100

	seq, cleanup := isequencer.New(params, storage, iTime)
	defer cleanup()

	var nextOffset isequencer.PLogOffset
	prevOffset := initialOffset
	prevNumber := initialNumber
	countOfFlushes := 0
	flushCalled := -1
	// Run 100 transactions
	const cycles = 100
	for i := 0; i < cycles; i++ {
		// Start transaction
		nextOffset = isequencer.WaitForStart(t, seq, 1, 1, true)
		nextNumber, err := seq.Next(1)
		require.NoError(err, "Failed to get next value in cycle %d", i)

		// Check out offset and number in dependence of Flush or Actualize was called
		switch flushCalled {
		case 1:
			// If Flush was called, check if offset and number are incremented
			require.Equal(nextOffset, prevOffset+1, "PLog offset should be incremented by 1 after Flush")
			require.Equal(nextNumber, prevNumber+1, "Sequence number should be incremented by 1 after Flush")
		case 0:
			// If Actualize was called, check if number and offset remain the same
			require.Equal(nextOffset, prevOffset, "PLog offset should not be incremented after Actualize")
			require.Equal(nextNumber, prevNumber, "Sequence number should not be incremented after Actualize")
		}

		// Finish transaction via Flush or Actualize
		// Randomly choose between Flush and Actualize (with equal probability)
		flushCalled = 0
		if rand.Int()%2 == 0 {
			seq.Flush()
			flushCalled = 1
			// Simulate write to pLog as CP does
			storage.AddPLogEntry(nextOffset, 1, 1, nextNumber)
			countOfFlushes++
			// Simulate some time passing to ensure that flusher finished
			iTime.Sleep(10 * time.Millisecond)
		} else {
			seq.Actualize()
		}
		// Update previous offset and number
		prevOffset = nextOffset
		prevNumber = nextNumber
	}
	// Check if the last offset in storage is equal to initial offset + count of flushes
	numbers, err := storage.ReadNumbers(1, []isequencer.SeqID{1})
	require.NoError(err)
	require.Equal(initialNumber+isequencer.Number(countOfFlushes), numbers[0], "Final number should be equal to initial number + count of flushes")
}

// [~server.design.sequences/test.isequencer.FlushPermanentlyFails~impl]
func TestISequencer_FlushPermanentlyFails(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	require := requirePkg.New(t)
	iTime := coreutils.NewITime()

	initialNumber := isequencer.Number(100)
	// Set up storage with initial values
	storage := isequencer.NewMockStorage(0, 0)
	storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
		isequencer.PLogOffset(1): {
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: initialNumber},
		},
	})

	params := createDefaultParams()
	params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
		1: {1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1},
	}
	params.MaxNumUnflushedValues = 5
	// Set up retry count for infinite retry
	params.RetryCount = 0

	seq, cleanup := isequencer.New(params, storage, iTime)
	defer cleanup()

	// Set up storage to simulate a permanent failure
	storage.SetWriteValuesAndOffset(errors.New("some error"))

	var num isequencer.Number
	var offset isequencer.PLogOffset
	var err error
	// first 6 sequence transaction will be ok
	// start 1st transaction for sequence = 1
	isequencer.WaitForStart(t, seq, 1, 1, true)

	num, err = seq.Next(isequencer.SeqID(1))
	require.NoError(err)
	require.Equal(initialNumber+1, num)

	seq.Flush()

	// start 5 more transactions for sequences from 2th to 6th
	for seqID := 2; seqID <= 6; seqID++ {
		isequencer.WaitForStart(t, seq, 1, 1, true)

		num, err = seq.Next(isequencer.SeqID(seqID))
		require.NoError(err)
		require.Equal(isequencer.Number(2), num)

		seq.Flush()
	}
	// The 7th transaction should fail
	offset = isequencer.WaitForStart(t, seq, 1, 1, false)
	require.Zero(offset)
	// Turn off the error simulation for correct cleanup
	storage.SetWriteValuesAndOffset(nil)
	// After the error has gone, the transaction should be ok
	offset = isequencer.WaitForStart(t, seq, 1, 1, true)
	require.Equal(isequencer.PLogOffset(7)+1, offset)
	// Next number for sequence = 1 should be initialNumber + 2
	num, err = seq.Next(isequencer.SeqID(1))
	require.NoError(err)
	require.Equal(initialNumber+isequencer.Number(2), num)
	// Next number for sequences from 2 for 6 should be 3, because we flushed numbers 2 for them before
	for i := 2; i <= 6; i++ {
		num, err = seq.Next(isequencer.SeqID(i))
		require.NoError(err)
		require.Equal(isequencer.Number(3), num)
	}
}

// [~server.design.sequences/test.isequencer.LongRecovery~impl]
func TestISequencer_LongRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := requirePkg.New(t)
	iTime := coreutils.NewITime()

	seqID_1 := isequencer.SeqID(1)
	seqID_2 := isequencer.SeqID(2)
	// Simulate a long recovery process gradually from 0 to 50 numEvents
	for numEvents := 1; numEvents <= 50; numEvents++ {
		// Fulfill pLog with data
		storage := isequencer.NewMockStorage(0, 0)

		pLogOffset := isequencer.PLogOffset(1)
		number := isequencer.Number(1)
		wsid := isequencer.WSID(1)
		for i := 0; i < numEvents; i++ {
			storage.AddPLogEntry(pLogOffset, wsid, seqID_1, number)
			number++
			storage.AddPLogEntry(pLogOffset, wsid, seqID_2, number)

			// we use one single seqID_1 for all wsids so numbers must grow monotonically
			number++
			pLogOffset++
		}

		params := createDefaultParams()
		params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
			1: {seqID_1: 1, seqID_2: 1},
		}
		seq, cleanup := isequencer.New(params, storage, iTime)

		nextOffset := isequencer.WaitForStart(t, seq, 1, wsid, true)
		require.Equal(isequencer.PLogOffset(numEvents)+1, nextOffset)

		num1, err := seq.Next(seqID_1)
		require.NoError(err)
		require.Equal(number-1, num1, numEvents)

		num2, err := seq.Next(seqID_2)
		require.NoError(err)
		require.Equal(number, num2)

		seq.Flush()

		// Simulate write to pLog as CP does
		storage.AddPLogEntry(nextOffset, wsid, seqID_1, num1)
		storage.AddPLogEntry(nextOffset, wsid, seqID_2, num2)

		// one more transaction after Flush

		nextOffset = isequencer.WaitForStart(t, seq, 1, wsid, true)
		require.Equal(isequencer.PLogOffset(numEvents)+2, nextOffset)

		num1, err = seq.Next(seqID_1)
		require.NoError(err)
		require.Equal(number, num1)

		num2, err = seq.Next(seqID_2)
		require.NoError(err)
		require.Equal(number+1, num2)

		seq.Actualize()

		// one more transaction after Actualize

		nextOffset = isequencer.WaitForStart(t, seq, 1, wsid, true)
		require.Equal(isequencer.PLogOffset(numEvents)+2, nextOffset)

		num1, err = seq.Next(seqID_1)
		require.NoError(err)
		require.Equal(number, num1)

		num2, err = seq.Next(seqID_2)
		require.NoError(err)
		require.Equal(number+1, num2)

		cleanup()
	}
}

// createDefaultStorage creates a storage with default configuration and common test data
func createDefaultStorage() *isequencer.MockStorage {
	storage := isequencer.NewMockStorage(0, 0)

	return storage
}

func createDefaultParams() isequencer.Params {
	return isequencer.NewDefaultParams(map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
		1: {1: 1},
	})
}
