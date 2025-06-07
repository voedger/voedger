/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isequencer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var actualizationTimeoutLimit = 1 * time.Second

// waitForActualization waits for actualization to complete by repeatedly calling Start
func WaitForStart(t *testing.T, seq ISequencer, wsKind WSKind, wsID WSID, shouldBeOk bool) PLogOffset {
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
	defer timeoutCtxCancel()

	offset, ok := seq.Start(wsKind, wsID)
	for !ok && timeoutCtx.Err() == nil {
		offset, ok = seq.Start(wsKind, wsID)
	}

	if shouldBeOk {
		require.True(t, ok)
	} else {
		require.False(t, ok)
	}

	return offset
}

// newMockStorage creates a new MockStorage instance
func NewMockStorage() *MockStorage {
	// notest
	return &MockStorage{
		Numbers:    make(map[WSID]map[SeqID]Number),
		NextOffset: PLogOffset(1),
		pLog:       make(map[PLogOffset][]SeqValue, 0),
	}
}

func (m *MockStorage) SetWriteValuesAndOffset(err error) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writeValuesAndOffsetError = err
}

func (m *MockStorage) SetReadNumbersError(err error) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReadNumbersError = err
}

func (m *MockStorage) SetReadNextPLogOffsetError(err error) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readNextOffsetError = err
}

func (m *MockStorage) SetOnWriteValuesAndOffset(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onWriteValuesAndOffset = f
}

func (m *MockStorage) SetOnActualizeFromPLog(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onActualizeFromPLog = f
}

// ReadNumbers implements isequencer.ISeqStorage.ReadNumbers
func (m *MockStorage) ReadNumbers(wsid WSID, seqIDs []SeqID) ([]Number, error) {
	// notest

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.onReadNumbers != nil {
		m.onReadNumbers()
	}

	if m.ReadNumbersError != nil {
		return nil, m.ReadNumbersError
	}

	result := make([]Number, len(seqIDs))

	wsNumbers, exists := m.Numbers[wsid]
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

// WriteValuesAndNextPLogOffset implements isequencer.ISeqStorage.WriteValuesAndNextPLogOffset
func (m *MockStorage) WriteValuesAndNextPLogOffset(batch []SeqValue, offset PLogOffset) error {
	// notest

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.onWriteValuesAndOffset != nil {
		m.onWriteValuesAndOffset()
	}

	if m.writeValuesAndOffsetError != nil {
		return m.writeValuesAndOffsetError
	}

	// batch could be empty here for the offset that is just written
	// case:
	// Start, Next, 1stFlush, flusher got the signal and gone to sleep before reading toBeFlushed,
	// Start, Next, 2ndFlush, flusher awake and write merged from both stransactions
	// on the next iteration flusher got 2nd signal and has nothing to write because everything is written already on 1st fire
	// that case is ok because batch is empty -> nothing will be written
	for _, entry := range batch {
		if _, ok := m.Numbers[entry.Key.WSID]; !ok {
			m.Numbers[entry.Key.WSID] = make(map[SeqID]Number)
		}
		m.Numbers[entry.Key.WSID][entry.Key.SeqID] = entry.Value
	}
	m.NextOffset = offset

	return nil
}

// ReadNextPLogOffset implements isequencer.ISeqStorage.ReadNextPLogOffset
func (m *MockStorage) ReadNextPLogOffset() (PLogOffset, error) {
	// notest

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.onReadNextPLogOffset != nil {
		m.onReadNextPLogOffset()
	}

	if m.readNextOffsetError != nil {
		return 0, m.readNextOffsetError
	}

	return m.NextOffset, nil
}

// ActualizeSequencesFromPLog implements isequencer.ISeqStorage.ActualizeSequencesFromPLog
func (m *MockStorage) ActualizeSequencesFromPLog(actualizerCtx context.Context, offset PLogOffset, batcher func(ctx context.Context, batch []SeqValue, offset PLogOffset) error) error {
	// notest
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.onActualizeFromPLog != nil {
		m.onActualizeFromPLog()
	}

	for offsetProbe := offset; offsetProbe < math.MaxInt; offsetProbe++ {
		batch, ok := m.pLog[offsetProbe]
		if !ok {
			// end of PLog is reached
			return nil
		}
		// Process entries in the mocked PLog from the provided offset
		select {
		case <-actualizerCtx.Done():
			return actualizerCtx.Err()
		default:
			// Continue processing
		}

		if err := batcher(actualizerCtx, batch, offsetProbe); err != nil {
			return err
		}
	}

	panic("impossible case")
}

// SetPLog sets the entire PLog contents for testing
func (m *MockStorage) SetPLog(plog map[PLogOffset][]SeqValue) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = plog
}

func (m *MockStorage) AddPLogEntry(offset PLogOffset, wsid WSID, seqID SeqID, number Number) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add the new entry to the PLog
	m.pLog[offset] = append(
		m.pLog[offset],
		SeqValue{
			Key:   NumberKey{WSID: wsid, SeqID: seqID},
			Value: number,
		},
	)
}
