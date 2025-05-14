/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/voedger/voedger/pkg/goutils/timeu"
)

func NewDefaultParams(seqTypes map[WSKind]map[SeqID]Number) Params {
	return Params{
		SeqTypes:                          seqTypes,
		MaxNumUnflushedValues:             DefaultMaxNumUnflushedValues,
		LRUCacheSize:                      DefaultLRUCacheSize,
		BatcherDelayOnToBeFlushedOverflow: defaultBatcherDelayOnToBeFlushedOverflow,
	}
}

// New creates a new sequencer
func New(params Params, seqStorage ISeqStorage, iTime timeu.ITime) (ISequencer, context.CancelFunc) {
	cache, err := lru.New[NumberKey, Number](params.LRUCacheSize)
	if err != nil {
		// notest
		panic("failed to create LRU cache: " + err.Error())
	}

	for _, seqIDsNumbers := range params.SeqTypes {
		for _, number := range seqIDsNumbers {
			if number < 1 {
				panic("initial numbers can not be <1")
			}
		}
	}

	cleanupCtx, cleanupCtxCancel := context.WithCancel(context.Background())
	s := &sequencer{
		params:                  params,
		cache:                   cache,
		toBeFlushed:             make(map[NumberKey]Number),
		inproc:                  make(map[NumberKey]Number),
		cleanupCtx:              cleanupCtx,
		cleanupCtxCancel:        cleanupCtxCancel,
		iTime:                   iTime,
		flusherCtxCancel:        func() {}, // flusher is not started -> prevent nil
		flusherSig:              make(chan struct{}, 1),
		actualizerWG:            &sync.WaitGroup{},
		seqStorage:              seqStorage,
		transactionIsInProgress: true, // to allow Actualize() to exec right below
	}
	s.Actualize()

	return s, s.cleanup
}
