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

type mockStorage struct {
	mu                       sync.RWMutex
	numbers                  map[WSID]map[SeqID]Number
	nextOffset               PLogOffset
	writeValuesError         error
	writeNextPLogOffsetError error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		numbers:    make(map[WSID]map[SeqID]Number),
		nextOffset: 0,
	}
}

func (m *mockStorage) ReadNumbers(wsid WSID, seqIDs []SeqID) ([]Number, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Number, len(seqIDs))
	if nums, exists := m.numbers[wsid]; exists {
		for i, id := range seqIDs {
			result[i] = nums[id] // Will be 0 if not found
		}
	}

	return result, nil
}

func (m *mockStorage) WriteValues(batch []SeqValue) error {
	if m.writeValuesError != nil {
		return m.writeValuesError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sv := range batch {
		wsNums, exists := m.numbers[sv.Key.WSID]
		if !exists {
			wsNums = make(map[SeqID]Number)
			m.numbers[sv.Key.WSID] = wsNums
		}
		// Only update if new value is greater
		if sv.Value > wsNums[sv.Key.SeqID] {
			wsNums[sv.Key.SeqID] = sv.Value
		}
	}

	return nil
}

func (m *mockStorage) WriteNextPLogOffset(nextOffset PLogOffset) error {
	if m.writeNextPLogOffsetError != nil {
		return m.writeNextPLogOffsetError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextOffset = nextOffset
	return nil
}

func (m *mockStorage) ReadNextPLogOffset() (PLogOffset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nextOffset, nil
}

func (m *mockStorage) ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func([]SeqValue, PLogOffset) error) error {
	return nil
}

func TestSequencer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Run("basic flow", func(t *testing.T) {
		mockedTime := coreutils.MockTime
		// Given
		storage := newMockStorage()
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

		// When
		offset, ok := seq.Start(1, 1)
		require.True(t, ok)
		require.Equal(t, PLogOffset(1), offset)

		// Generate new sequence numbers 100 times
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
		storage := newMockStorage()
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
	})
}

// FIXME: Fix the test
func TestBatcher(t *testing.T) {
	t.Run("should aggregate max values and write to storage", func(t *testing.T) {
		// Given
		storage := newMockStorage()
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
		storage := newMockStorage()
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
		storage := newMockStorage()
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

		storage.writeValuesError = errors.New("write error")
		// When
		err := seq.(*sequencer).batcher(batch, PLogOffset(1))

		// Then
		require.ErrorIs(t, err, coreutils.ErrRetryAttemptsExceeded)
	})
}
