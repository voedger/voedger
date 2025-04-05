/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"sync"

	lruPkg "github.com/hashicorp/golang-lru/v2"

	"github.com/voedger/voedger/pkg/coreutils"
)

func NewDefaultParams(seqTypes map[WSKind]map[SeqID]Number) Params {
	return Params{
		SeqTypes:              seqTypes,
		MaxNumUnflushedValues: DefaultMaxNumUnflushedValues,
		LRUCacheSize:          DefaultLRUCacheSize,
		RetryDelay:            defaultRetryDelay,
		RetryCount:            defaultRetryCount,
	}
}

// New creates a new sequencer
func New(params Params, seqStorage ISeqStorage, iTime coreutils.ITime) (ISequencer, context.CancelFunc) {
	lru, err := lruPkg.New[NumberKey, Number](params.LRUCacheSize)
	if err != nil {
		// notest
		panic("failed to create LRU cache: " + err.Error())
	}

	cleanupCtx, cleanupCtxCancel := context.WithCancel(context.Background())
	s := &sequencer{
		params:           params,
		cache:            lru,
		toBeFlushed:      make(map[NumberKey]Number),
		inproc:           make(map[NumberKey]Number),
		cleanupCtx:       cleanupCtx,
		cleanupCtxCancel: cleanupCtxCancel,
		iTime:            iTime,
		flusherStartedCh: make(chan struct{}, 1),
		flusherSig:       make(chan struct{}, 1),
		actualizerWG:     &sync.WaitGroup{},
		seqStorage:       seqStorage,
	}
	s.actualizerInProgress.Store(false)

	// Instance has actualizer() goroutine started.
	s.startActualizer()
	s.actualizerWG.Wait()

	return s, s.cleanup
}
