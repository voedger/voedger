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

	iTime coreutils.ITime
}
