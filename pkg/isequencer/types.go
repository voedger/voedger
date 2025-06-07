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
	"github.com/voedger/voedger/pkg/goutils/timeu"
)

type PartitionID uint16
type SeqID uint16  // istructs.QNameID
type WSKind uint16 // istructs.QNameID
type WSID uint64
type Number uint64
type PLogOffset uint64
type ClusterAppID = uint32

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

	MaxNumUnflushedValues int // 500
	// Size of the LRU cache, NumberKey -> Number.
	LRUCacheSize                      int           // 100_000
	BatcherDelayOnToBeFlushedOverflow time.Duration // 5 * time.Millisecond
}

// sequencer implements ISequencer
// [~server.design.sequences/cmp.sequencer~impl]
type sequencer struct {
	params     Params
	seqStorage ISeqStorage

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
	// Is not accessed concurrently since
	// - Is accessed by actualizer() and cleanup()
	// - cleanup() first shutdowns the actualizer() then flusher()	flusherWg sync.WaitGroup
	flusherWG sync.WaitGroup
	// Buffered channel [1] to signal flusher to flush
	// Written (non-blocking) by Flush()
	flusherSig chan struct{}

	// To be flushed
	// cleared by actualizer()
	toBeFlushed map[NumberKey]Number
	// Will be 6 if the offset of the last processed event is 4
	toBeFlushedOffset PLogOffset
	// Protects toBeFlushed and toBeFlushedOffset
	toBeFlushedMu sync.RWMutex

	// Written by Next()
	inproc map[NumberKey]Number

	iTime timeu.ITime

	transactionIsInProgress bool
}

// [~server.design.sequences/test.isequencer.mockISeqStorage~impl]
// MockStorage implements ISeqStorage for testing purposes
type MockStorage struct {
	Numbers                   map[WSID]map[SeqID]Number
	NextOffset                PLogOffset
	ReadNumbersError          error
	writeValuesAndOffsetError error
	mu                        sync.RWMutex
	pLog                      map[PLogOffset][]SeqValue // Simulated PLog entries
	readNextOffsetError       error
	onWriteValuesAndOffset    func()
	onReadNextPLogOffset      func()
	onActualizeFromPLog       func()
	onReadNumbers             func()
}

type SequencesTrustLevel byte
