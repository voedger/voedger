/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer_test

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/isequencer"
)

func TestComplex(t *testing.T) {
	// t.Skip()
	for range 100000 {
		TestISequencer_ComplexEvents(t)
	}
}

func TestISequencer_ComplexEvents(t *testing.T) {
	require := require.New(t)

	const (
		numWSID  = 5
		numSeqID = 8
	)

	cases := []struct {
		name                        string
		plog                        map[isequencer.PLogOffset][]isequencer.SeqValue
		expectedNextNumbersOverride isequencer.TExpectedNumbers // to make expected number for a certain wsid and seqID be not 1
		initialExpectedOffset       isequencer.PLogOffset
	}{
		// {
		// 	name:                  "empty plog",
		// 	plog:                  map[isequencer.PLogOffset][]isequencer.SeqValue{},
		// 	initialExpectedOffset: 1,
		// },
		// {
		// 	name: "1 simple event no CUDs",
		// 	plog: map[isequencer.PLogOffset][]isequencer.SeqValue{
		// 		1: {},
		// 	},
		// 	initialExpectedOffset: 2,
		// },
		// {
		// 	name: "1 simple event 1 CUD",
		// 	plog: map[isequencer.PLogOffset][]isequencer.SeqValue{
		// 		1: {{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 1}},
		// 	},
		// 	expectedNextNumbersOverride: expectedNumbers{1: {1: 2}},
		// 	initialExpectedOffset:       2,
		// },
		{
			name: "few events",
			plog: map[isequencer.PLogOffset][]isequencer.SeqValue{
				1: {
					{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 10},
					{Key: isequencer.NumberKey{WSID: 1, SeqID: 2}, Value: 11},
					{Key: isequencer.NumberKey{WSID: 2, SeqID: 3}, Value: 12},
					{Key: isequencer.NumberKey{WSID: 2, SeqID: 4}, Value: 13},
				},
				2: {},
				3: {
					{Key: isequencer.NumberKey{WSID: 3, SeqID: 5}, Value: 14},
					{Key: isequencer.NumberKey{WSID: 3, SeqID: 6}, Value: 15},
					{Key: isequencer.NumberKey{WSID: 4, SeqID: 7}, Value: 16},
					{Key: isequencer.NumberKey{WSID: 4, SeqID: 8}, Value: 17},
				},
			},
			initialExpectedOffset: 4,
			expectedNextNumbersOverride: isequencer.TExpectedNumbers{
				1: {1: 11, 2: 12},
				2: {3: 13, 4: 14},
				3: {5: 15, 6: 16},
				4: {7: 17, 8: 18},
				// 3: {5: 15},
			},
		},
	}

	params := createDefaultParams()

	// set known seqIDs and initial numbers
	seqIDsNumbers := map[isequencer.SeqID]isequencer.Number{}
	for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
		seqIDsNumbers[seqID] = 1
	}
	params.SeqTypes = map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{1: seqIDsNumbers}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			storage := createDefaultStorage()
			storage.SetPLog(c.plog)
			seq, cleanup := isequencer.New(params, storage, coreutils.MockTime)
			logger.Info("new")
			defer cleanup()

			// set expected next numbers for all seqIDs to 1
			isequencer.ExpectedNumbers = getExpectedNumbers(numWSID, numSeqID)

			// apply expected numbers from the test case
			for wsid, seqIDs := range isequencer.ExpectedNumbers {
				for seqID := range seqIDs {
					if overrideNumber, ok := c.expectedNextNumbersOverride[wsid][seqID]; ok {
						seqIDs[seqID] = overrideNumber
					}
				}
			}

			expectedOffset := c.initialExpectedOffset

			// for each wsid: Start -> Next for each SeqID -> Flush -> Start -> Next for each SeqID -> Actualize -> Start -> Next for each SeqID -> Flush
			for wsid := isequencer.WSID(1); wsid <= numWSID; wsid++ {
				plogOffset := isequencer.WaitForStart(t, seq, 1, wsid, true)
				logger.Info("waitforstart")
				require.Equal(expectedOffset, plogOffset, "wsid", wsid)

				// 1st transaction - check expected next numbers + flush
				for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
					num, err := seq.Next(seqID)
					logger.Info("next")
					require.NoError(err)
					if isequencer.ExpectedNumbers[wsid][seqID] != num {
						isequencer.StacksMU.Lock()
						// log.Println(isequencer.Stacks[len(isequencer.Stacks)-1])
						isequencer.StacksMU.Unlock()
					}
					require.Equal(isequencer.ExpectedNumbers[wsid][seqID], num, strconv.Itoa(int(wsid)), strconv.Itoa(int(seqID)))
				}
				seq.Flush()
				logger.Info("flush")

				// simulate write to plog as CP does
				for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
					storage.AddPLogEntry(plogOffset, wsid, seqID, isequencer.ExpectedNumbers[wsid][seqID])
				}

				// 2nd transaction - check expected next numbers+1, then Actualize
				plogOffset = isequencer.WaitForStart(t, seq, 1, wsid, true)
				logger.Info("waitforstart")
				require.Equal(expectedOffset+1, plogOffset, "wsid", wsid)
				for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
					num, err := seq.Next(seqID)
					logger.Info("next")
					require.NoError(err)
					require.Equal(isequencer.ExpectedNumbers[wsid][seqID]+1, num)
				}
				seq.Actualize()
				logger.Info("actualize")

				// 3rd transaction - check expected next numbers+1 again, then Flush
				plogOffset = isequencer.WaitForStart(t, seq, 1, wsid, true)
				logger.Info("waitforstart")
				require.Equal(expectedOffset+1, plogOffset)
				for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
					num, err := seq.Next(seqID)
					logger.Info("next")
					require.NoError(err)
					require.Equal(isequencer.ExpectedNumbers[wsid][seqID]+1, num)
				}
				seq.Flush()
				logger.Info("flush")

				// simulate write to plog as CP does
				for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
					storage.AddPLogEntry(plogOffset, wsid, seqID, isequencer.ExpectedNumbers[wsid][seqID])
				}

				expectedOffset += 2 // after 2 Flush()'es
			}
		})
	}
}

func TestISequencer_Start(t *testing.T) {
	require := require.New(t)
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

		offset := isequencer.WaitForStart(t, sequencer, 1, 1, true)
		require.NotZero(t, offset)

		require.Panics(func() {
			sequencer.Start(1, 1)
		})
	})

	t.Run("should reject when actualization in progress", func(t *testing.T) {
		storage := createDefaultStorage()
		stuckActualizationCh := make(chan any)
		seq, cleanup := isequencer.New(params, storage, iTime)
		defer cleanup()

		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(isequencer.PLogOffset(1), offset)

		// force actualization to stuck to guarantee that Start will be rejected
		storage.SetOnActualizeFromPLog(func() {
			<-stuckActualizationCh
		})
		defer close(stuckActualizationCh)

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
			isequencer.WaitForStart(t, sequencer, 2, 1, true) // WSKind 2 is not defined
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
		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
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
		offset = isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(isequencer.PLogOffset(3), offset)

		// Verify we can get the next number in sequence
		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(100+count+1), num, "Sequence should continue from last value")
	})

	t.Run("correct offset after Flush without Next", func(t *testing.T) {
		iTime := coreutils.MockTime
		storage := createDefaultStorage()
		storage.SetPLog(map[isequencer.PLogOffset][]isequencer.SeqValue{
			isequencer.PLogOffset(1): {
				{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		params := createDefaultParams()
		seq, cancel := isequencer.New(params, storage, iTime)
		defer cancel()

		// First transaction
		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(isequencer.PLogOffset(2), offset)

		seq.Flush()

		offset = isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(isequencer.PLogOffset(3), offset)

	})
}

func TestISequencer_Flush(t *testing.T) {
	t.Run("should correctly increment sequence values after flush", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := require.New(t)

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
		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
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
		offset = isequencer.WaitForStart(t, seq, 1, 1, true)
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
		require := require.New(t)

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
		require := require.New(t)
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
		offset := isequencer.WaitForStart(t, sequencer, 1, 1, true)
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
	require := require.New(t)
	t.Run("should return incremented sequence number", func(t *testing.T) {
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

		isequencer.WaitForStart(t, sequencer, 1, 1, true)

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

		isequencer.WaitForStart(t, sequencer, 1, 1, true)

		num, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(initialValue+1, num, "Next should use initial value when sequence not in storage")

		sequencer.Flush()
	})

	t.Run("should panic when called without starting transaction", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := createDefaultStorage()
		sequencer, cancel := isequencer.New(createDefaultParams(), storage, iTime)
		defer cancel()

		require.Panics(func() {
			sequencer.Next(1)
		}, "Next should panic when called without starting a transaction")
	})

	t.Run("should handle multiple sequence types correctly", func(t *testing.T) {
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

		isequencer.WaitForStart(t, sequencer, 1, 1, true)

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
		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.NotZero(offset)

		num, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(301), num, "Sequence should continue from last value")

		seq.Flush()

		// Transaction 2
		offset = isequencer.WaitForStart(t, seq, 1, 1, true)
		require.NotZero(offset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(302), num, "Sequence should continue from last value")

		seq.Flush()
	})

	t.Run("should handle cache eviction correctly", func(t *testing.T) {
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

	t.Run("unknown SeqID -> error", func(t *testing.T) {
		iTime := coreutils.MockTime
		storage := createDefaultStorage()
		params := createDefaultParams()
		seq, cancel := isequencer.New(params, storage, iTime)
		defer cancel()
		isequencer.WaitForStart(t, seq, 1, 1, true)
		num, err := seq.Next(10)
		require.ErrorIs(err, isequencer.ErrUnknownSeqID)
		require.Zero(num)
	})
}

func TestISequencer_Actualize(t *testing.T) {
	require := require.New(t)
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

	t.Run("Actualize() during actualize -> panic", func(t *testing.T) {
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
		offset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(isequencer.PLogOffset(2), offset)

		seq.Actualize()
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
		require.Equal(isequencer.Number(1), num)

		// Actualize with empty PLog
		seq.Actualize()

		// Should be able to start a new transaction
		nextOffset := isequencer.WaitForStart(t, seq, 1, 1, true)
		require.Equal(initialOffset, nextOffset)

		num, err = seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(1), num)

		num2, err := seq.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(2), num2)
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
	require := require.New(t)
	iTime := coreutils.NewITime()

	initialNumber := isequencer.Number(100)
	initialOffset := isequencer.PLogOffset(1)
	// Set up storage with initial values
	storage := isequencer.NewMockStorage()
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
	flushCalled := false
	// Run 100 transactions
	const cycles = 100
	for i := 0; i < cycles; i++ {
		// Start transaction
		nextOffset = isequencer.WaitForStart(t, seq, 1, 1, true)
		nextNumber, err := seq.Next(1)
		require.NoError(err, "Failed to get next value in cycle %d", i)

		// Check out offset and number in dependence of Flush or Actualize was called
		if i == 0 || flushCalled {
			// If Flush was called, check if offset and number are incremented
			require.Equal(nextOffset, prevOffset+1, "PLog offset should be incremented by 1 after Flush")
			require.Equal(nextNumber, prevNumber+1, "Sequence number should be incremented by 1 after Flush")

		} else if !flushCalled {
			// If Actualize was called, check if number and offset remain the same
			require.Equal(nextOffset, prevOffset, "PLog offset should not be incremented after Actualize")
			require.Equal(nextNumber, prevNumber, "Sequence number should not be incremented after Actualize")
		}

		// Finish transaction via Flush or Actualize
		// Randomly choose between Flush and Actualize (with equal probability)
		if rand.Int()%2 == 0 {
			seq.Flush()
			flushCalled = true
			// Simulate write to pLog as CP does
			storage.AddPLogEntry(nextOffset, 1, 1, nextNumber)
			countOfFlushes++
		} else {
			flushCalled = false
			seq.Actualize()
		}
		// Update previous offset and number
		prevOffset = nextOffset
		prevNumber = nextNumber
	}
	// wait for last flush be accomplished
	isequencer.WaitForStart(t, seq, 1, 1, true)
	lastNum, err := seq.Next(1)
	require.NoError(err)
	require.Equal(initialNumber+isequencer.Number(countOfFlushes)+1, lastNum, "next number after cycles should be equal to initial number + count of flushes + 1")
}

// [~server.design.sequences/test.isequencer.FlushPermanentlyFails~impl]
func TestISequencer_FlushPermanentlyFails(t *testing.T) {
	require := require.New(t)
	iTime := coreutils.NewITime()

	initialNumber := isequencer.Number(100)
	// Set up storage with initial values
	storage := isequencer.NewMockStorage()
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
		require.Equal(isequencer.Number(1), num)

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
		require.Equal(isequencer.Number(2), num)
	}
}

// [~server.design.sequences/test.isequencer.LongRecovery~impl]
func TestISequencer_LongRecovery(t *testing.T) {
	const (
		maxNumEvents = 50
		seqID_1      = isequencer.SeqID(1)
		seqID_2      = isequencer.SeqID(2)
	)
	require := require.New(t)
	iTime := coreutils.NewITime()

	// Simulate a long recovery process gradually from 0 to 50 numEvents
	for numEvents := 1; numEvents <= maxNumEvents; numEvents++ {
		// Fulfill pLog with data
		storage := isequencer.NewMockStorage()

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

func TestNewExecutesActualize(t *testing.T) {
	iTime := coreutils.NewITime()
	storage := isequencer.NewMockStorage()
	actualizationStartedCh := make(chan any)
	storage.SetOnActualizeFromPLog(func() {
		close(actualizationStartedCh)
	})

	pLogOffset := isequencer.PLogOffset(1)
	number := isequencer.Number(1)
	wsid := isequencer.WSID(1)
	storage.AddPLogEntry(pLogOffset, wsid, 1, number)

	params := createDefaultParams()
	_, cleanup := isequencer.New(params, storage, iTime)
	defer cleanup()
	<-actualizationStartedCh
}

func TestLongRecovery_ForceRace(t *testing.T) {
	// t.Skip()
	err := new(exec.PipedExec).Command(
		"go",
		"test",
		"-run=^TestISequencer_LongRecovery$",
		"-count=1",
		"-race",
	).Run(os.Stdout, os.Stderr)
	require.NoError(t, err)
}

// createDefaultStorage creates a storage with default configuration and common test data
func createDefaultStorage() *isequencer.MockStorage {
	storage := isequencer.NewMockStorage()

	return storage
}

func createDefaultParams() isequencer.Params {
	return isequencer.NewDefaultParams(map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
		1: {1: 1},
	})
}

// type expectedNumbers map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number

func getExpectedNumbers(numWSID isequencer.WSID, numSeqID isequencer.SeqID) isequencer.TExpectedNumbers {
	res := isequencer.TExpectedNumbers{}
	for wsid := isequencer.WSID(1); wsid <= numWSID; wsid++ {
		for seqID := isequencer.SeqID(1); seqID <= numSeqID; seqID++ {
			wsidSeqs, ok := res[wsid]
			if !ok {
				wsidSeqs = map[isequencer.SeqID]isequencer.Number{}
				res[wsid] = wsidSeqs
			}
			wsidSeqs[seqID] = 1
		}
	}
	return res
}
