/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	lruPkg "github.com/hashicorp/golang-lru/v2"

	"github.com/voedger/voedger/pkg/coreutils"
)

type PartitionID uint16
type SeqID uint16  // QNameID
type WSKind uint16 // QNameID
type WSID uint64
type Number uint64
type PLogOffset uint64

type NumberKey struct {
	WSID  WSID
	SeqID SeqID
}

type SeqValue struct {
	Key   NumberKey
	Value Number
}

// Params for the ISequencer implementation.
type Params struct {

	// Sequences and their initial values.
	// Only these sequences are managed by the sequencer (ref. ErrUnknownSeqID).
	SeqTypes map[WSKind]map[SeqID]Number

	SeqStorage ISeqStorage

	MaxNumUnflushedValues int           // 500
	MaxFlushingInterval   time.Duration // 500 * time.Millisecond
	// Size of the LRU cache, NumberKey -> Number.
	LRUCacheSize int // 100_000
}

type sequencer struct {
	params *Params

	actualizerInProgress atomic.Bool
	// actualizerCtxCancel is used by cleanup() function
	actualizerCtxCancel context.CancelFunc
	actualizerWG        *sync.WaitGroup

	lru *lruPkg.Cache[NumberKey, Number]

	// Initialized by Start()
	// Example:
	// - 4 is the last processed event
	// - nextOffset keeps 5
	// - Start() returns 5 and increments nextOffset to 6
	nextOffset PLogOffset

	// If Sequencing Transaction is in progress then currentWSID has non-zero value.
	currentWSID   WSID
	currentWSKind WSKind

	// If cleanupCtx is Done, then actualization should stop immediately
	cleanupCtx       context.Context
	cleanupCtxCancel context.CancelFunc

	// Closes when flusher needs to be stopped
	flusherCtxCancel context.CancelFunc
	// Used to wait for flusher goroutine to exit
	// Set to nil when flusher is not running
	// Is not accessed concurrently since
	// - Is accessed by actualizer() and cleanup()
	// - cleanup() first shutdowns the actualizer() then flusher()	flusherWg sync.WaitGroup
	flusherWG *sync.WaitGroup
	// Used in tests to signal that flusher is started
	flusherStartedCh chan struct{}
	// Buffered channel [1] to signal flusher to flush
	// Written (non-blocking) by Flush()
	flusherSig chan struct{}
	// sync mechanism to prevent multiple flusher goroutines
	flusherInProgress atomic.Bool

	// To be flushed
	toBeFlushed map[NumberKey]Number
	// Will be 6 if the offset of the last processed event is 4
	toBeFlushedOffset PLogOffset
	// Protects toBeFlushed and toBeFlushedOffset
	toBeFlushedMu sync.RWMutex

	// Written by Next()
	inproc   map[NumberKey]Number
	inprocMu sync.RWMutex
	// need to check if Flush or Actualize was called after previous Start
	transactionIsInProgress atomic.Bool

	iTime coreutils.ITime
}

// MockStorage implements ISeqStorage for testing purposes
type MockStorage struct {
	mu                        sync.RWMutex
	Numbers                   map[WSID]map[SeqID]Number
	nextOffset                PLogOffset
	pLog                      [][]SeqValue // Simulated PLog entries
	readNextOffsetError       error
	readNumbersError          error
	writeValuesAndOffsetError error
}

// newMockStorage creates a new MockStorage instance
func NewMockStorage() *MockStorage {
	return &MockStorage{
		Numbers: make(map[WSID]map[SeqID]Number),
		pLog:    make([][]SeqValue, 0),
	}
}

// ReadNumbers implements isequencer.ISeqStorage.ReadNumbers
func (m *MockStorage) ReadNumbers(wsid WSID, seqIDs []SeqID) ([]Number, error) {
	if m.readNumbersError != nil {
		return nil, m.readNumbersError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

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

// WriteValues implements isequencer.ISeqStorage.WriteValuesAndOffset
func (m *MockStorage) WriteValuesAndOffset(batch []SeqValue, offset PLogOffset) error {
	if m.writeValuesAndOffsetError != nil {
		return m.writeValuesAndOffsetError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sv := range batch {
		wsNumbers, exists := m.Numbers[sv.Key.WSID]
		if !exists {
			wsNumbers = make(map[SeqID]Number)
			m.Numbers[sv.Key.WSID] = wsNumbers
		}
		wsNumbers[sv.Key.SeqID] = sv.Value
	}

	m.nextOffset = offset

	return nil
}

// ReadNextPLogOffset implements isequencer.ISeqStorage.ReadNextPLogOffset
func (m *MockStorage) ReadNextPLogOffset() (PLogOffset, error) {
	if m.readNextOffsetError != nil {
		return 0, m.readNextOffsetError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.nextOffset, nil
}

// ActualizeSequencesFromPLog implements isequencer.ISeqStorage.ActualizeSequencesFromPLog
func (m *MockStorage) ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func(batch []SeqValue, offset PLogOffset) error) error {
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
		currentOffset := PLogOffset(i)
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
func (m *MockStorage) AddPLogEntry(batch []SeqValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = append(m.pLog, batch)
}

// SetPLog sets the entire PLog contents for testing
func (m *MockStorage) SetPLog(plog [][]SeqValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = plog
}

// ClearPLog removes all entries from the mock PLog
func (m *MockStorage) ClearPLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = make([][]SeqValue, 0)
}
