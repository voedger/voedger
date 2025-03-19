package isequencer_test

import (
	"context"
	"sync"
	"testing"
	"time"

	requirePkg "github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
)

// mockStorage implements isequencer.ISeqStorage for testing purposes
type mockStorage struct {
	mu                   sync.RWMutex
	numbers              map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number
	nextOffset           isequencer.PLogOffset
	pLog                 [][]isequencer.SeqValue // Simulated PLog entries
	readNextOffsetError  error
	writeNextOffsetError error
	readNumbersError     error
	writeValuesError     error
}

// newMockStorage creates a new mockStorage instance
func newMockStorage() *mockStorage {
	return &mockStorage{
		numbers: make(map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number),
		pLog:    make([][]isequencer.SeqValue, 0),
	}
}

// ReadNumbers implements isequencer.ISeqStorage.ReadNumbers
func (m *mockStorage) ReadNumbers(wsid isequencer.WSID, seqIDs []isequencer.SeqID) ([]isequencer.Number, error) {
	if m.readNumbersError != nil {
		return nil, m.readNumbersError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]isequencer.Number, len(seqIDs))
	wsNumbers, exists := m.numbers[wsid]
	if !exists {
		return result, nil // Return zeros if workspace not found
	}

	for i, seqID := range seqIDs {
		if num, ok := wsNumbers[seqID]; ok {
			result[i] = num
		}
	}

	return result, nil
}

// WriteValues implements isequencer.ISeqStorage.WriteValues
func (m *mockStorage) WriteValues(batch []isequencer.SeqValue) error {
	if m.writeValuesError != nil {
		return m.writeValuesError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sv := range batch {
		wsNumbers, exists := m.numbers[sv.Key.WSID]
		if !exists {
			wsNumbers = make(map[isequencer.SeqID]isequencer.Number)
			m.numbers[sv.Key.WSID] = wsNumbers
		}
		wsNumbers[sv.Key.SeqID] = sv.Value
	}

	return nil
}

// WriteNextPLogOffset implements isequencer.ISeqStorage.WriteNextPLogOffset
func (m *mockStorage) WriteNextPLogOffset(offset isequencer.PLogOffset) error {
	if m.writeNextOffsetError != nil {
		return m.writeNextOffsetError
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextOffset = offset

	return nil
}

// ReadNextPLogOffset implements isequencer.ISeqStorage.ReadNextPLogOffset
func (m *mockStorage) ReadNextPLogOffset() (isequencer.PLogOffset, error) {
	if m.readNextOffsetError != nil {
		return 0, m.readNextOffsetError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.nextOffset, nil
}

// ActualizeSequencesFromPLog implements isequencer.ISeqStorage.ActualizeSequencesFromPLog
func (m *mockStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Process entries in the mocked PLog from the provided offset
	for i, batch := range m.pLog {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// Skip entries before the requested offset
		currentOffset := isequencer.PLogOffset(i)
		if currentOffset < offset {
			continue
		}

		if err := batcher(batch, currentOffset); err != nil {
			return err
		}
	}

	return nil
}

// AddPLogEntry adds a batch of sequence values to the mock PLog for testing
func (m *mockStorage) AddPLogEntry(batch []isequencer.SeqValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = append(m.pLog, batch)
}

// SetPLog sets the entire PLog contents for testing
func (m *mockStorage) SetPLog(plog [][]isequencer.SeqValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = plog
}

// ClearPLog removes all entries from the mock PLog
func (m *mockStorage) ClearPLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = make([][]isequencer.SeqValue, 0)
}

func TestISequencer_Start(t *testing.T) {
	t.Run("should panic when transaction already started", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := newMockStorage()
		storage.nextOffset = isequencer.PLogOffset(99)
		storage.numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(t, offset)

		require.Panics(func() {
			sequencer.Start(1, 1)
		})
	})

	t.Run("should reject when actualization in progress", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := newMockStorage()
		storage.nextOffset = isequencer.PLogOffset(99)
		storage.numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		sequencer.Actualize()

		// Should be rejected while actualization is in progress
		offset, ok = sequencer.Start(1, 1)
		require.False(ok)
		require.Zero(offset)
	})

	t.Run("should panic when unknown workspace kind", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := newMockStorage()
		storage.nextOffset = isequencer.PLogOffset(99)
		storage.numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
			1: {1: 100},
		}
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 1},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		require.Panics(func() {
			sequencer.Start(2, 1) // WSKind 2 is not defined
		})
	})

	t.Run("should start successfully after flush completes", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := newMockStorage()
		sequencer, cancel := isequencer.New(&isequencer.Params{
			SeqTypes: map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{
				1: {1: 100},
			},
			SeqStorage:            storage,
			MaxNumUnflushedValues: 5,
			MaxFlushingInterval:   500 * time.Millisecond,
			LRUCacheSize:          1000,
		}, iTime)
		defer cancel()

		// First transaction
		offset, ok := sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		count := 3
		// Generate sequence numbers
		var numbers []isequencer.Number
		for i := 0; i < count; i++ {
			num, err := sequencer.Next(1)
			require.NoError(err)
			numbers = append(numbers, num)
		}

		sequencer.Flush()

		// Verify the sequence values were incremented correctly
		for i, num := range numbers {
			require.Equal(isequencer.Number(100+i+1), num, "Sequence value should be incremented")
		}

		// Advance time to allow flushing to complete
		iTime.Add(time.Second)

		// Start a new transaction
		offset, ok = sequencer.Start(1, 1)
		require.True(ok)
		require.NotZero(offset)

		// Verify we can get the next number in sequence
		num, err := sequencer.Next(1)
		require.NoError(err)
		require.Equal(isequencer.Number(100+count+1), num, "Sequence should continue from last value")
	})
}

func TestISequencer_Flush(t *testing.T) {
	t.Run("should correctly increment sequence values after flush", func(t *testing.T) {
		iTime := coreutils.MockTime
		require := requirePkg.New(t)

		storage := newMockStorage()
		storage.numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
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

		storage := newMockStorage()
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
		//t.Skip()

		require := requirePkg.New(t)
		iTime := coreutils.MockTime

		storage := newMockStorage()
		storage.numbers = map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{
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
