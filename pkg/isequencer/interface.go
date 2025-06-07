/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
)

type ISeqStorage interface {

	// If number is not found, returns 0
	// Each Number matches its SeqID by array index
	ReadNumbers(WSID, []SeqID) ([]Number, error)

	// IDs in batch.Values are unique
	// len(batch) may be 0
	// offset: Next offset to be used
	// batch MUST be written first, then offset
	WriteValuesAndNextPLogOffset(batch []SeqValue, offset PLogOffset) error

	ReadNextPLogOffset() (PLogOffset, error)

	// ActualizeSequencesFromPLog scans PLog from the given offset and send values to the batcher.
	// Values are sent per event, unordered, ISeqValue.Keys are not unique.
	ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func(ctx context.Context, batch []SeqValue, offset PLogOffset) error) error
}

type IVVMSeqStorageAdapter interface {
	GetNumber(appID ClusterAppID, wsid WSID, seqID SeqID) (ok bool, number Number, err error)
	GetPLogOffset(partitionID PartitionID) (ok bool, pLogOffset PLogOffset, err error)
	PutPLogOffset(partitionID PartitionID, plogOffset PLogOffset) error
	PutNumbers(appID ClusterAppID, batch []SeqValue) error
}

// ISequencer defines the interface for working with sequences.
// ISequencer methods must not be called concurrently.
// Use: { Start {Next} ( Flush | Actualize ) }
//
// Definitions
// - Sequencing Transaction: Start -> Next -> (Flush | Actualize)
// - Actualization: Making the persistent state of the sequences consistent with the PLog.
// - Flushing: Writing the accumulated sequence values to the storage.
// - LRU Cache: Least Recently Used cache that keep the most recent next sequence values in memory.
// [~server.design.sequences/cmp.ISequencer~impl]
type ISequencer interface {

	// Start starts Sequencing Transaction for the given WSID.
	// Marks Sequencing Transaction as in progress.
	// Panics if Sequencing Transaction is already started.
	// Normally returns the next PLogOffset, true
	// Returns `0, false` if:
	// - Actualization is in progress
	// - The number of unflushed values exceeds the maximum threshold
	// If ok is true, the caller must call Flush() or Actualize() to complete the Sequencing Transaction.
	// [~server.design.sequences/cmp.ISequencer.Start~impl]
	Start(wsKind WSKind, wsID WSID) (plogOffset PLogOffset, ok bool)

	// Next returns the next sequence number for the given SeqID.
	// Panics if Sequencing Transaction is not in progress.
	// err: ErrUnknownSeqID if the sequence is not defined in Params.SeqTypes.
	Next(seqID SeqID) (num Number, err error)

	// Flush completes Sequencing Transaction.
	// Panics if Sequencing Transaction is not in progress.
	// Sends the current batch to the flushing queue and completes the event processing.
	Flush()

	// Actualize cancels Sequencing Transaction and starts the Actualization process.
	// Panics if Actualization is already in progress.
	// Panics if Sequencing Transaction is not in progress.
	// Flow:
	// - Mark Sequencing Transaction as not in progress
	// - Cancel and wait Flushing
	// - Empty LRU Cache
	// - Do Actualization process
	// - Write next PLogOffset
	Actualize()
}
