/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import "context"

type ISeqStorage interface {

	// If number is not found, returns 0
	ReadNumbers(WSID, []SeqID) ([]Number, error)

	// IDs in batch.Values are unique
	// Values must be written first, then Offset
	WriteValues(batch []SeqValue) error

	// Next offset to be used
	WriteNextPLogOffset(offset PLogOffset) error
	ReadNextPLogOffset() (PLogOffset, error)

	// ActualizeSequencesFromPLog scans PLog from the given offset and send values to the batcher.
	// Values are sent per event, unordered, ISeqValue.Keys are not unique.
	ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func(batch []SeqValue, offset PLogOffset) error) error
}

// ISequencer defines the interface for working with sequences.
// ISequencer methods must not be called concurrently.
// Use: { Start {Next} ( Flush | Actualize ) }
//
// Definitions
// - Sequencing Transaction: Start -> Next -> (Flush | Actualize)
// - Actualization: Making the persistent state of the sequences consistent with the PLog.
// - Flushing: Writing the accumulated sequence values to the storage.
// - LRU: Least Recently Used cache that keep the most recent next sequence values in memory.
type ISequencer interface {

	// Start starts Sequencing Transaction for the given WSID.
	// Marks Sequencing Transaction as in progress.
	// Panics if Sequencing Transaction is already started.
	// Normally returns the next PLogOffset, true
	// Returns `0, false` if:
	// - Actualization is in progress
	// - The number of unflushed values exceeds the maximum threshold
	// If ok is true, the caller must call Flush() or Actualize() to complete the Sequencing Transaction.
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
	// - Empty LRU
	// - Do Actualization process
	// - Write next PLogOffset
	Actualize()
}
