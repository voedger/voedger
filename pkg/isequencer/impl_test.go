/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
)

func TestSequencer(t *testing.T) {
	require := require.New(t)

	t.Run("basic flow", func(t *testing.T) {
		mockedTime := coreutils.MockTime
		// Given
		storage := NewMockStorage()
		storage.SetPLog(map[PLogOffset][]SeqValue{
			PLogOffset(1): {
				{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 100},
			},
		})

		params := NewDefaultParams(map[WSKind]map[SeqID]Number{
			1: {1: 1},
		})

		seq, cleanup := New(params, storage, mockedTime)

		// When
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(2), offset)

		// Generate new sequence Numbers 100 times
		for i := 1; i <= 100; i++ {
			num, err := seq.Next(1)
			require.NoError(err)
			require.Equal(Number(100+i), num)
		}

		seq.Flush()

		cleanup()
		seq.(*sequencer).flusherWG.Wait()

		nums, err := storage.ReadNumbers(1, []SeqID{1})
		require.NoError(err)
		require.Equal(Number(200), nums[0])
	})
}

func TestSequencer_Start(t *testing.T) {
	require := require.New(t)
	t.Run("should reject when too many unflushed values + allow after Flush", func(t *testing.T) {
		iTime := coreutils.MockTime

		storage := NewMockStorage()
		params := NewDefaultParams(map[WSKind]map[SeqID]Number{
			1: {1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1},
		})
		params.MaxNumUnflushedValues = 5
		waitForFlushCh := make(chan any)
		writeOnFlushOccuredCh := make(chan any)
		storage.onWriteValuesAndOffset = func() {
			close(writeOnFlushOccuredCh)
			<-waitForFlushCh
		}
		seq, cancel := New(params, storage, iTime)
		defer cancel()

		offset, ok := seq.Start(1, 1)
		require.True(ok)
		require.Equal(PLogOffset(1), offset)

		// obtain 6 numbers, 6th should overflow toBeFlushed
		for i := 0; i < 6; i++ {
			num, err := seq.Next(SeqID(i + 1))
			require.NoError(err)
			require.Equal(Number(1), num)
		}

		seq.Flush()

		<-writeOnFlushOccuredCh

		// failed to start because toBeFlushed is overflowed
		offset, ok = seq.Start(1, 1)
		require.False(ok)
		require.Zero(offset)
		close(waitForFlushCh)
	})
}

func TestBatcher(t *testing.T) {
	t.Run("should aggregate max values and wait for unflushed values threshold", func(t *testing.T) {
		require := require.New(t)

		// Given
		ctx := context.Background()
		storage := NewMockStorage()
		mockTime := coreutils.MockTime
		params := NewDefaultParams(nil)
		seq, cleanup := New(params, storage, mockTime)
		defer cleanup()
		s := seq.(*sequencer)

		// Set up the batch to be processed
		batch := []SeqValue{
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 104},
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 103},
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102},
			{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102},
			{Key: NumberKey{WSID: 1, SeqID: 2}, Value: 201},
			{Key: NumberKey{WSID: 1, SeqID: 4}, Value: 401},
			{Key: NumberKey{WSID: 1, SeqID: 4}, Value: 402},
			{Key: NumberKey{WSID: 2, SeqID: 1}, Value: 52},
			{Key: NumberKey{WSID: 2, SeqID: 1}, Value: 51},
		}

		err := s.batcher(ctx, batch, 6)
		require.NoError(err)

		// Verify toBeFlushed has been updated with maximum values
		s.toBeFlushedMu.RLock()
		defer s.toBeFlushedMu.RUnlock()

		// Should now have values from batch with max values preserved
		require.Equal(Number(104), s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}])
		require.Equal(Number(201), s.toBeFlushed[NumberKey{WSID: 1, SeqID: 2}])
		require.Equal(Number(402), s.toBeFlushed[NumberKey{WSID: 1, SeqID: 4}])
		require.Equal(Number(52), s.toBeFlushed[NumberKey{WSID: 2, SeqID: 1}])
		require.Len(s.toBeFlushed, 4)

		// Verify offset was updated
		require.Equal(PLogOffset(7), s.toBeFlushedOffset, "Should update toBeFlushedOffset to offset + 1")

		// Verify nextOffset was updated
		require.Equal(PLogOffset(7), s.nextOffset, "Should update nextOffset to offset + 1")
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		storage := NewMockStorage()
		params := NewDefaultParams(map[WSKind]map[SeqID]Number{
			1: {1: 100},
		})

		// Small threshold to force waiting
		params.MaxNumUnflushedValues = 1

		seq, cleanup := New(params, storage, coreutils.MockTime)
		defer cleanup()
		s := seq.(*sequencer)

		// Set up the batch to be processed
		batch := []SeqValue{{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102}}

		// simulate toBeFlushed is fulfilled
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1

		// make the context be canceled
		cancel()
		err := s.batcher(ctx, batch, 6)
		require.ErrorIs(err, context.Canceled)
	})

	t.Run("continue after toBeFlushed drain", func(t *testing.T) {
		require := require.New(t)
		storage := NewMockStorage()
		params := NewDefaultParams(map[WSKind]map[SeqID]Number{
			1: {1: 100},
		})

		// Small threshold to force waiting
		params.MaxNumUnflushedValues = 1
		mockTime := coreutils.NewMockTime()
		seq, cleanup := New(params, storage, mockTime)
		defer cleanup()
		s := seq.(*sequencer)

		// Set up the batch to be processed
		batch := []SeqValue{{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 102}}

		// simulate toBeFlushed is fulfilled
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1

		mockTime.FireNextTimerImmediately()
		mockTime.SetOnNextNewTimerChan(func() {
			// simulate toBeFlushed is drained by flusher
			s.toBeFlushed = map[NumberKey]Number{}
		})
		err := s.batcher(context.Background(), batch, 6)
		require.NoError(err)
	})
}

func TestContextCloseDuringStorageErrors(t *testing.T) {
	require := require.New(t)

	storageErr := errors.New("storage error")
	mockTime := coreutils.MockTime

	params := NewDefaultParams(map[WSKind]map[SeqID]Number{
		1: {1: 1},
	})

	t.Run("flusher()", func(t *testing.T) {
		storage := NewMockStorage()
		seq, _ := New(params, storage, mockTime)
		s := seq.(*sequencer)
		storage.SetWriteValuesAndOffset(storageErr)
		defer func() { storage.SetReadNextPLogOffsetError(nil) }()
		triedToWriteCh := make(chan any)
		storage.onWriteValuesAndOffset = func() {
			s.cleanupCtxCancel()
			close(triedToWriteCh)
		}

		// Test with empty values
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1

		// simulate normal sequencer behaviour
		s.signalToFlushing()

		<-triedToWriteCh
		s.flusherWG.Wait()

	})

	t.Run("flushValues()", func(t *testing.T) {
		storage := NewMockStorage()
		seq, cleanup := New(params, storage, mockTime)
		s := seq.(*sequencer)
		storage.SetWriteValuesAndOffset(storageErr)
		defer func() { storage.SetReadNextPLogOffsetError(nil) }()
		storage.onWriteValuesAndOffset = func() {
			cleanup() // ctx is closed here
		}

		// Test with empty values
		s.toBeFlushed[NumberKey{WSID: 1, SeqID: 1}] = 1
		err := s.flushValues(PLogOffset(1))
		// closed ctx causes Retry() returned immediately after storage error

		require.ErrorIs(err, context.Canceled)
	})

	t.Run("actualizer()", func(t *testing.T) {
		t.Run("on ReadNextPLogOffset", func(t *testing.T) {
			storage := NewMockStorage()
			seq, cleanup := New(params, storage, mockTime)
			storage.SetReadNextPLogOffsetError(storageErr)
			s := seq.(*sequencer)
			storage.onReadNextPLogOffset = func() {
				cleanup() // ctx is closed here
			}
			s.actualizerWG.Add(1) // simulate s.Actualize() behaviour
			s.actualizer(s.cleanupCtx)
		})

		t.Run("on ActualizeSequencesFromPLog", func(t *testing.T) {
			storage := NewMockStorage()
			storage.SetPLog(map[PLogOffset][]SeqValue{PLogOffset(1): {{Key: NumberKey{WSID: 1, SeqID: 1}, Value: 100}}})
			seq, cleanup := New(params, storage, mockTime)
			s := seq.(*sequencer)
			storage.onActualizeFromPLog = func() {
				cleanup() // ctx is closed here
			}
			s.actualizerWG.Add(1) // simulate s.Actualize() behaviour
			s.actualizer(s.cleanupCtx)
		})
	})

	t.Run("Next()", func(t *testing.T) {
		storage := NewMockStorage()
		seq, cleanup := New(params, storage, mockTime)
		s := seq.(*sequencer)
		storage.SetReadNumbersError(storageErr)
		storage.onReadNumbers = func() {
			cleanup() // ctx is closed here
		}
		_, ok := s.Start(1, 1)
		require.True(ok)
		num, err := s.Next(1)
		require.ErrorIs(err, context.Canceled)
		require.Zero(num)
	})

}

func TestNextNumberSourceOrder(t *testing.T) {
	require := require.New(t)

	// Set up mock storage and sequencer
	storage := NewMockStorage()
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

	params := NewDefaultParams(map[WSKind]map[SeqID]Number{
		1: {1: 1},
	})

	seq, cleanup := New(params, storage, mockTime)
	defer cleanup()

	numberKey := NumberKey{WSID: 1, SeqID: 1}

	t.Run("check the value is cached after next", func(t *testing.T) {
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)
		numInitial, err := seq.Next(1)
		require.NoError(err)
		require.NotZero(numInitial)
		numCached, ok := seq.(*sequencer).cache.Get(numberKey)
		require.True(ok)
		require.Equal(numInitial, numCached)

		seq.Actualize()
	})

	t.Run("check taken from cache on next", func(t *testing.T) {
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)

		// tamper the cache to ensure we'll use cache on Next
		seq.(*sequencer).cache.Add(numberKey, 10001)

		// expect read from cache first on normal Next call
		numFromCache, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(10002), numFromCache)

		seq.Actualize()
	})

	t.Run("missing in cache -> take from inproc", func(t *testing.T) {
		// start
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)

		// fill the cache
		numInitial, err := seq.Next(1)
		require.NoError(err)
		require.NotZero(numInitial)

		// evict the cached number
		require.True(seq.(*sequencer).cache.Remove(numberKey))

		// tamper inproc to be sure we'll read exactly from inproc in this case
		seq.(*sequencer).inproc[numberKey] = 20001

		// missing in cache -> expect read from inproc
		numActual, err := seq.Next(1)
		require.NoError(err)
		require.Equal(Number(20002), numActual)

		seq.Actualize()
	})

	t.Run("missing in cache and in inproc -> take from toBeFlushed", func(t *testing.T) {
		// start
		offset := WaitForStart(t, seq, 1, 1, true)
		require.Equal(PLogOffset(6), offset)

		// fill the cache and inproc
		numInitial, err := seq.Next(1)
		require.NoError(err)
		require.NotZero(numInitial)

		// clear inproc + keep toBeFlushed filled by making flusher() stuck
		continueCh := make(chan any)
		writeValuesAndOffsetCh := make(chan any)
		storage.onWriteValuesAndOffset = func() {
			close(writeValuesAndOffsetCh)
			<-continueCh
		}
		defer func() {
			storage.onWriteValuesAndOffset = nil
		}()
		seq.Flush()
		seq.(*sequencer).inproc = map[NumberKey]Number{}

		// clear cache
		removed := seq.(*sequencer).cache.Remove(numberKey)
		require.True(removed)

		// tamper toBeFlushed to ensure we'll read exactly from toBeFlushed in this case
		<-writeValuesAndOffsetCh // avoid data race on toBeFlushed
		seq.(*sequencer).toBeFlushed[numberKey] = 30001

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

func TestWrongCacheSize(t *testing.T) {
	require.Panics(t, func() { New(Params{LRUCacheSize: -1}, nil, nil) })
}

func TestPanicOnWrongInitialNumber(t *testing.T) {
	require.Panics(t, func() {
		New(Params{LRUCacheSize: DefaultLRUCacheSize, SeqTypes: map[WSKind]map[SeqID]Number{1: {1: 0}}}, nil, nil)
	})
}
