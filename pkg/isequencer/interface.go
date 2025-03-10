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

	WritePLogOffset(offset PLogOffset) error

	// Last offset successfully written by WriteValues
	ReadLastWrittenPLogOffset() (PLogOffset, error)

	// Scan PLog from the given offset and send values to the batcher.
	// Values are sent per event, unordered, IDs are not unique.
	// Batcher is responsible for batching, ordering, and ensuring uniqueness, and uses ISeqStorage.WriteValues.
	// Batcher can block the execution for some time, but it terminates if the ctx is done.
	ActualizePLog(ctx context.Context, offset PLogOffset, batcher func(batch []SeqValue, offset PLogOffset) error) error
}

// ISequencer methods must not be called concurrently.
// Use: { Start {Next} ( Flush | Actualize ) }
type ISequencer interface {

	// Starts event processing for the given WSID.
	// Normal flow: increments the current PLogOffset value and returns this value with `true`.
	// Panics if event processing is already started.
	// Returns `false` if:
	// - Actualization is in progress
	// - The number of unflushed values exceeds the maximum threshold
	// If ok is true, the caller must call Flush() or Actualize() to complete the event processing.
	Start(wsKind WSKind, wsID WSID) (plogOffset PLogOffset, ok bool)

	// Returns the next sequence number for the given SeqID.
	// If seqID is unknown, panics.
	// err: ErrUnknownSeqID
	Next(seqID SeqID) (num Number, err error)

	// Finishes event processing.
	// Panics if event processing is not in progress.
	// Sends the current batch to the flushing queue and completes the event processing.
	Flush()

	// Finishes event processing.
	// Panics if actualization is already in progress.
	// Panics if event processing is not in progress.
	// Completes event processing.
	// If flusher() is running, stops and waits for it.
	// Starts actualizer().
	Actualize()
}
