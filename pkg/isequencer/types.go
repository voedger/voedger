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

type PartitionID uint64
type SeqID uint16
type WSKind uint16
type WSID uint16
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

	// If cleanupCtx is Done, then actualization should stop immediately
	cleanupCtx       context.Context
	cleanupCtxCancel context.CancelFunc
	// Closes when flusher needs to be stopped
	flusherCtx       context.Context
	flusherCtxCancel context.CancelFunc
	// Used to wait for flusher goroutine to exit
	flusherWg            sync.WaitGroup
	actualizerInProgress atomic.Bool
	actualizerWg         sync.WaitGroup

	lru *lruPkg.Cache[NumberKey, Number]

	// To be flushed
	toBeFlushed       map[NumberKey]Number
	toBeFlushedOffset PLogOffset
	// Protects toBeFlushed and toBeFlushedOffset
	toBeFlushedMu sync.RWMutex

	// Written by Next()
	inproc       map[NumberKey]Number
	inprocMu     sync.RWMutex
	inprocOffset PLogOffset

	// Initialized by Start()
	currentWSID   WSID
	currentWSKind WSKind

	iTime coreutils.ITime
}
