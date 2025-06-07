#### interface.go

```go
/*
 * Copyright (c) 2025-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package isequencer

import (
	"context"
	"time"
)

type SeqID      uint16
type WSKind     uint16
type WSID       uint64
type Number     uint64
type PLogOffset uint64

type NumberKey struct {
	WSID  WSID
	SeqID SeqID
}

type SeqValue struct {
	Key    NumberKey
	Value  Number
}

// To be injected into the ISequencer implementation.
//
type ISeqStorage interface {

  // If number is not found, returns 0
	ReadNumbers(WSID, []SeqID) ([]Number, error)
  ReadNextPLogOffset() (PLogOffset, error)

	// IDs in batch.Values are unique
  // len(batch) may be 0
  // offset: Next offset to be used
  // batch MUST be written first, then offset
	WriteValuesAndNextPLogOffset(batch []SeqValue, nextPLogOffset PLogOffset) error

  // ActualizeSequencesFromPLog scans PLog from the given offset and send values to the batcher.
	// Values are sent per event, unordered, ISeqValue.Keys are not unique.
  // err: ctx.Err() if ctx is closed
	ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func(ctx context.Context, batch []SeqValue, offset PLogOffset) error) (err error)
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
  Flush()

  // Actualize cancels Sequencing Transaction and starts the Actualization process.
  // Panics if Actualization is already in progress.
    // Flow:
  // - Mark Sequencing Transaction as not in progress
  // - Do Actualization process
  //   - Cancel and wait Flushing
  //   - Empty LRU
  //   - Write next PLogOffset
  Actualize()

}

// Params for the ISequencer implementation.
type Params struct {

	// Sequences and their initial values.
	// Only these sequences are managed by the sequencer (ref. ErrUnknownSeqID).
	SeqTypes map[WSKind]map[SeqID]Number

	MaxNumUnflushedValues int           // 500
	// Size of the LRU cache, NumberKey -> Number.
	LRUCacheSize int // 100_000

  BatcherDelayOnToBeFlushedOverflow time.Duration // 5 * time.Millisecond
}
```

#### Implementation requirements

```go
// filepath: pkg/isequencer: impl.go

import (
  "context"
  "time"

  "github.com/hashicorp/golang-lru/v2"
)

// Implements isequencer.ISequencer
// Keeps next (not current) values in LRU and type ISeqStorage interface
type sequencer struct {
  params *Params

  actualizerInProgress atomic.Bool

  // Set by s.Actualize(), never cleared (zeroed).
  // Used by s.cleanup().
  actualizerCtxCancel context.CancelFunc
  actualizerWG  *sync.WaitGroup


  // Cleared by s.Actualize()
  lru *lru.Cache

  // Initialized by Start()
  // Example:
  // - 4 is the offset ofthe last event in the PLog
  // - nextOffset keeps 5
  // - Start() returns 5 and increments nextOffset to 6
  nextOffset PLogOffset

  // If Sequencing Transaction is in progress then currentWSID has non-zero value.
  // Cleared by s.Actualize()
  currentWSID   WSID
  currentWSKind WSKind

  // Set by s.actualizer()
  // Closed when flusher needs to be stopped
  flusherCtxCancel context.CancelFunc
  // Used to wait for flusher goroutine to exit
  // Set to nil when flusher is not running
  // Is not accessed concurrently since
  // - Is accessed by actualizer() and cleanup()
  // - cleanup() first shutdowns the actualizer() then flusher()
  flusherWG  *sync.WaitGroup
  // Buffered channel [1] to signal flusher to flush
  // Written (non-blocking) by Flush()
  flusherSig chan struct{}

  // To be flushed
  toBeFlushed map[NumberKey]Number
  // Will be 6 if the offset of the last processed event is 4
  toBeFlushedOffset PLogOffset
  // Protects toBeFlushed and toBeFlushedOffset
  toBeFlushedMu sync.RWMutex

  // Written by Next()
  inproc map[NumberKey]Number

}

// New creates a new instance of the Sequencer type.
// Instance has actualizer() goroutine started.
// cleanup: function to stop the actualizer.
func New(isequencer.Params, ISeqStorage, timeu.ITime) (isequencer.ISequencer, cleanup func()) {
  // ...
}

// Flush implements isequencer.ISequencer.Flush.
// Flow:
//   Copy s.inproc and s.nextOffset to s.toBeFlushed and s.toBeFlushedOffset
//   Clear s.inproc
//   Increase s.nextOffset
//   Non-blocking write to s.flusherSig
func (s *sequencer) Flush() {
  // ...
}

// Next implements isequencer.ISequencer.Next.
// It ensures thread-safe access to sequence values and handles various caching layers.
//
// Flow:
// - Validate equencing Transaction status
// - Get initialValue from s.params.SeqTypes and ensure that SeqID is known
// - Try to obtain the next value using:
//   - Try s.lru (can be evicted)
//   - Try s.inproc
//   - Try s.toBeFlushed (use s.toBeFlushedMu to synchronize)
//   - Try s.params.SeqStorage.ReadNumber()
//      - Read all known numbers for wsKind, wsID
//        - If number is 0 then initial value is used
//      - Write all numbers to s.lru
// - Write value+1 to s.lru
// - Write value+1 to s.inproc
// - Return value
func (s *sequencer) Next(seqID SeqID) (num Number, err error) {
  // ...
}

// batcher processes a batch of sequence values and writes maximum values to storage.
//
// Flow:
// - Wait until len(s.toBeFlushed) < s.params.MaxNumUnflushedValues
//   - Lock/Unlock
//   - Sleep for s.params.BatcherDelayOnToBeFlushedOverflow
//   - check ctx (return ctx.Err())
// - s.nextOffset = offset + 1
// - Store maxValues in s.toBeFlushed: max Number for each SeqValue.Key
// - s.toBeFlushedOffset = offset + 1
//
func (s *sequencer) batcher(ctx ctx.Context, values []SeqValue, offset PLogOffset) (err error) {
  // ...
}

// Actualize implements isequencer.ISequencer.Actualize.
// Flow:
// - Validate Actualization status (s.actualizerInProgress is false)
// - Set s.actualizerInProgress to true
// - Set s.actualizerCtxCancel, s.actualizerWG
// - Start the actualizer() goroutine
func (s *sequencer) Actualize() {
  // ...
}

/*
actualizer is started in goroutine by Actualize().

Flow:

- if s.flusherWG is not nil
  - s.cancelFlusherCtx()
  - Wait for s.flusherWG
  - s.flusherWG = nil
- Clean s.lru, s.nextOffset, s.currentWSID, s.currentWSKind, s.toBeFlushed, s.inproc, s.toBeFlushedOffset
- s.flusherWG, s.flusherCtxCancel + start flusher() goroutine
- Read nextPLogOffset from s.params.SeqStorage.ReadNextPLogOffset()
- Use s.params.SeqStorage.ActualizeSequencesFromPLog() and s.batcher()

ctx handling:
 - if ctx is closed exit

Error handling:
- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) actualizer(ctx context.Context) {
  // ...
}

/*
flusher is started in goroutine by actualizer().

Flow:

- Wait for ctx.Done() or s.flusherSig
- if ctx.Done() exit
- Lock s.toBeFlushedMu
- Copy s.toBeFlushedOffset to flushOffset (local variable)
- Copy s.toBeFlushed to flushValues []SeqValue (local variable)
- Unlock s.toBeFlushedMu
- s.params.SeqStorage.WriteValues(flushValues, flushOffset)
- Lock s.toBeFlushedMu
- for each key in flushValues remove key from s.toBeFlushed if values are the same
- Unlock s.toBeFlushedMu

Error handling:

- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) flusher(ctx context.Context) {
  // ...
}

// cleanup stops the actualizer() and flusher() goroutines.
// Flow:
// - if s.actualizerInProgress
//   - s.cancelActualizerCtx()
//   - Wait for s.actualizerWG
// - if s.flusherWG is not nil
//   - s.cancelFlusherCtx()
//   - Wait for s.flusherWG
//   - s.flusherWG = nil
func (s *sequencer) cleanup() {
  // ...
}
```
