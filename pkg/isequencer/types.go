/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import "time"

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
