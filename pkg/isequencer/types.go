/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"math"
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

	MaxNumUnflushedValues int // 500
	// Size of the LRU cache, NumberKey -> Number.
	LRUCacheSize int           // 100_000
	BatcherDelay time.Duration // 5 * time.Millisecond
}

// sequencer implements ISequencer
// [~server.design.sequences/cmp.sequencer~impl]
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

// [~server.design.sequences/test.isequencer.mockISeqStorage~impl]
// MockStorage implements ISeqStorage for testing purposes
type MockStorage struct {
	Numbers                   map[WSID]map[SeqID]Number
	NextOffset                PLogOffset
	ReadNumbersError          error
	writeValuesAndOffsetError error
	mu                        sync.RWMutex
	numbersMu                 sync.RWMutex
	pLog                      map[PLogOffset][]SeqValue // Simulated PLog entries
	readNextOffsetError       error
	readTimeout               time.Duration
	writeTimeout              time.Duration
	onWriteValuesAndOffset    func()
}

// newMockStorage creates a new MockStorage instance
func NewMockStorage(readTimeout, writeTimeout time.Duration) *MockStorage {
	// notest
	return &MockStorage{
		Numbers:      make(map[WSID]map[SeqID]Number),
		NextOffset:   PLogOffset(1),
		pLog:         make(map[PLogOffset][]SeqValue, 0),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
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

// ReadNumbers implements isequencer.ISeqStorage.ReadNumbers
func (m *MockStorage) ReadNumbers(wsid WSID, seqIDs []SeqID) ([]Number, error) {
	// notest
	if m.readTimeout > 0 {
		time.Sleep(m.readTimeout)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ReadNumbersError != nil {
		return nil, m.ReadNumbersError
	}

	result := make([]Number, len(seqIDs))

	m.numbersMu.RLock()
	defer m.numbersMu.RUnlock()

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
	if m.writeTimeout > 0 {
		time.Sleep(m.writeTimeout)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.writeValuesAndOffsetError != nil {
		return m.writeValuesAndOffsetError
	}

	if m.onWriteValuesAndOffset != nil {
		m.onWriteValuesAndOffset()
	}

	m.numbersMu.Lock()
	defer m.numbersMu.Unlock()

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
	if m.readTimeout > 0 {
		time.Sleep(m.readTimeout)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.readNextOffsetError != nil {
		return 0, m.readNextOffsetError
	}

	return m.NextOffset, nil
}

// ActualizeSequencesFromPLog implements isequencer.ISeqStorage.ActualizeSequencesFromPLog
func (m *MockStorage) ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func(ctx context.Context, batch []SeqValue, offset PLogOffset) error) error {
	// notest
	m.mu.RLock()
	defer m.mu.RUnlock()

	for offsetProbe := offset; offsetProbe < math.MaxInt; offsetProbe++ {
		batch, ok := m.pLog[offsetProbe]
		if !ok {
			// end of PLog is reached
			return nil
		}
		// Process entries in the mocked PLog from the provided offset
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		if m.writeTimeout > 0 {
			time.Sleep(m.writeTimeout)
		}

		if err := batcher(ctx, batch, offsetProbe); err != nil {
			return err
		}
	}

	return nil
}

// SetPLog sets the entire PLog contents for testing
func (m *MockStorage) SetPLog(plog map[PLogOffset][]SeqValue) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = plog
}

func (m *MockStorage) AddPLogEntry(offset, wsid int, seqID SeqID, number int) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add the new entry to the PLog
	m.pLog[PLogOffset(offset)] = append( //nolint:gosec
		m.pLog[PLogOffset(offset)], //nolint:gosec
		SeqValue{
			Key:   NumberKey{WSID: WSID(wsid), SeqID: seqID}, //nolint:gosec
			Value: Number(number),                            //nolint:gosec
		},
	)
}

// ClearPLog removes all entries from the mock PLog
func (m *MockStorage) ClearPLog() {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = make(map[PLogOffset][]SeqValue)
}
