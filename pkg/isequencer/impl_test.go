/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
)

type mockStorage struct {
	mu      sync.RWMutex
	numbers map[WSID]map[SeqID]Number
	offset  PLogOffset
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		numbers: make(map[WSID]map[SeqID]Number),
		offset:  0,
	}
}

func (m *mockStorage) ReadNumbers(wsid WSID, seqIDs []SeqID) ([]Number, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Number, len(seqIDs))
	if nums, exists := m.numbers[wsid]; exists {
		for i, id := range seqIDs {
			result[i] = nums[id]
		}
	}
	return result, nil
}

func (m *mockStorage) WriteValues(batch []SeqValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sv := range batch {
		if m.numbers[sv.Key.WSID] == nil {
			m.numbers[sv.Key.WSID] = make(map[SeqID]Number)
		}
		m.numbers[sv.Key.WSID][sv.Key.SeqID] = sv.Value
	}
	return nil
}

func (m *mockStorage) WritePLogOffset(offset PLogOffset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offset = offset
	return nil
}

func (m *mockStorage) ReadLastWrittenPLogOffset() (PLogOffset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.offset, nil
}

func (m *mockStorage) ActualizePLog(ctx context.Context, offset PLogOffset, batcher func([]SeqValue, PLogOffset) error) error {
	return nil
}

func TestSequencer(t *testing.T) {
	mockedTime := coreutils.NewITime()

	t.Run("basic flow", func(t *testing.T) {
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

		seq, cleanup := NewSequencer(params)
		defer cleanup()

		// When
		offset, ok := seq.Start(1, 1)
		require.True(t, ok)
		require.Equal(t, PLogOffset(1), offset)

		num, err := seq.Next(1)
		require.NoError(t, err)
		require.Equal(t, Number(101), num)

		seq.Flush()

		// Then
		mockedTime.Sleep(1 * time.Second)

		nums, err := storage.ReadNumbers(1, []SeqID{1})
		require.NoError(t, err)
		require.Equal(t, Number(101), nums[0])
	})

	t.Run("actualization", func(t *testing.T) {
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

		seq, cleanup := NewSequencer(params)
		defer cleanup()

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
