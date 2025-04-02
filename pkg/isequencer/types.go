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

type sequencer struct {
	params *Params

	actualizerInProgress atomic.Bool
	// actualizerCtxCancel is used by cleanup() function
	actualizerCtxCancel context.CancelFunc
	actualizerWG        *sync.WaitGroup

	cache *lruPkg.Cache[NumberKey, Number]

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
	NextOffset                PLogOffset
	pLog                      map[PLogOffset][]SeqValue // Simulated PLog entries
	readNextOffsetError       error
	ReadNumbersError          error
	WriteValuesAndOffsetError error
	readTimeout               time.Duration
	writeTimeout              time.Duration
	onWriteValuesAndOffset    func()
}

// newMockStorage creates a new MockStorage instance
func NewMockStorage(readTimeout, writeTimeout time.Duration) *MockStorage {
	// notest
	return &MockStorage{
		pLog:         make(map[PLogOffset][]SeqValue, 0),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// ReadNumbers implements isequencer.ISeqStorage.ReadNumbers
func (m *MockStorage) ReadNumbers(wsid WSID, seqIDs []SeqID) ([]Number, error) {
	// notest
	if m.readTimeout > 0 {
		time.Sleep(m.readTimeout)
	}

	if m.ReadNumbersError != nil {
		return nil, m.ReadNumbersError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	numbers := make([]Number, len(seqIDs))
	for i := m.NextOffset; i > PLogOffset(0); i-- {
		if _, ok := m.pLog[i]; !ok {
			continue
		}

		for _, seqValue := range m.pLog[i] {
			for j, seqID := range seqIDs {
				if numbers[j] != 0 {
					continue // Skip if already found
				}

				if seqValue.Key.SeqID == seqID && seqValue.Key.WSID == wsid {
					numbers[j] = seqValue.Value
					break
				}
			}
		}
	}

	return numbers, nil
}

// WriteValues implements isequencer.ISeqStorage.WriteValuesAndOffset
func (m *MockStorage) WriteValuesAndOffset(batch []SeqValue, offset PLogOffset) error {
	// notest
	if m.writeTimeout > 0 {
		time.Sleep(m.writeTimeout)
	}

	if m.WriteValuesAndOffsetError != nil {
		return m.WriteValuesAndOffsetError
	}

	if m.onWriteValuesAndOffset != nil {
		m.onWriteValuesAndOffset()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.pLog[offset]; ok {
		panic("WriteValuesAndOffset: offset already exists")
	}

	m.pLog[offset] = batch
	m.NextOffset = offset

	return nil
}

// ReadNextPLogOffset implements isequencer.ISeqStorage.ReadNextPLogOffset
func (m *MockStorage) ReadNextPLogOffset() (PLogOffset, error) {
	// notest
	if m.readTimeout > 0 {
		time.Sleep(m.readTimeout)
	}

	if m.readNextOffsetError != nil {
		return 0, m.readNextOffsetError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.pLog) == 0 {
		return m.NextOffset, nil
	}

	// Find the maximum offset in the PLog
	maxOffset := PLogOffset(0)
	for offset := range m.pLog {
		if offset > maxOffset {
			maxOffset = offset
		}
	}

	// If the maximum offset is greater than the current NextOffset, update it
	if maxOffset > m.NextOffset {
		m.NextOffset = maxOffset
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

func (m *MockStorage) AddPLogEntry(offset, wsid, seqID, number int) {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.pLog[PLogOffset(offset)]; ok {
		panic("AddPLogEntry: offset already exists")
	}
	// Add the new entry to the PLog
	m.pLog[PLogOffset(offset)] = []SeqValue{
		{
			Key:   NumberKey{WSID: WSID(wsid), SeqID: SeqID(seqID)},
			Value: Number(number),
		},
	}
}

// ClearPLog removes all entries from the mock PLog
func (m *MockStorage) ClearPLog() {
	// notest
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pLog = make(map[PLogOffset][]SeqValue)
}
