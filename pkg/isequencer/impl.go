/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/voedger/voedger/pkg/coreutils"
)

// Start starts Sequencing Transaction for the given WSID.
// Marks Sequencing Transaction as in progress.
// Panics if Sequencing Transaction is already started.
// Normally returns the next PLogOffset, true
// Returns `0, false` if:
// - Actualization is in progress
// - The number of unflushed values exceeds the maximum threshold
// If ok is true, the caller must call Flush() or Actualize() to complete the Sequencing Transaction.
// [~server.design.sequences/cmp.sequencer.Start~impl]
func (s *sequencer) Start(wsKind WSKind, wsID WSID) (plogOffset PLogOffset, ok bool) {
	// Check if cleanup is in progress
	if s.cleanupCtx.Err() != nil {
		panic("sequencer is in cleanup state")
	}

	// Check if Actualization is in progress
	if s.actualizerInProgress.Load() {
		return 0, false
	}

	if s.transactionIsInProgress {
		panic("Sequencing Transaction is already started")
	}

	// Verify wsKind exists in supported types
	if _, exists := s.params.SeqTypes[wsKind]; !exists {
		panic("unknown wsKind")
	}

	// Check unflushed values threshold
	s.toBeFlushedMu.RLock()
	// The number of unflushed values exceeds the maximum threshold
	if len(s.toBeFlushed) > s.params.MaxNumUnflushedValues {
		s.toBeFlushedMu.RUnlock()
		s.signalToFlushing()

		return 0, false
	}
	s.toBeFlushedMu.RUnlock()

	s.currentWSID = wsID
	s.currentWSKind = wsKind
	s.transactionIsInProgress = true

	return s.nextOffset, true
}

func (s *sequencer) startFlusher() {
	flusherCtx, flusherCtxCancel := context.WithCancel(context.Background())
	s.flusherCtxCancel = flusherCtxCancel
	s.flusherWG.Add(1)
	go s.flusher(flusherCtx)
}

func (s *sequencer) startActualizer() {
	actualizerCtx, actualizerCtxCancel := context.WithCancel(s.cleanupCtx)
	s.actualizerCtxCancel = actualizerCtxCancel

	s.actualizerWG.Add(1)
	go s.actualizer(actualizerCtx)
}

func (s *sequencer) stopFlusher() {
	s.flusherCtxCancel()
	s.flusherWG.Wait()
	select {
	case <-s.flusherSig:
	default:
	}
}

// signalToFlushing is used to signal the flusher to start flushing.
func (s *sequencer) signalToFlushing() {
	//  Non-blocking write to s.flusherSig
	select {
	case s.flusherSig <- struct{}{}:
		// notest
	default:
		// notest
	}
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
- s.params.SeqStorage.WriteValuesAndNextPLogOffset(flushValues, flushOffset)
- Lock s.toBeFlushedMu
- for each key in flushValues remove key from s.toBeFlushed if values are the same
- Unlock s.toBeFlushedMu

Error handling:

- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) flusher(flusherCtx context.Context) {
	defer s.flusherWG.Done()

	// Wait for ctx.Done()
	for flusherCtx.Err() == nil {
		select {
		case <-flusherCtx.Done():
			return
		// Wait for s.flusherSig
		case <-s.flusherSig:
		}

		var flushOffset PLogOffset
		s.toBeFlushedMu.RLock()
		flushValues := make([]SeqValue, 0, len(s.toBeFlushed))
		{
			flushOffset = s.toBeFlushedOffset
			if flushOffset == 0 {
				s.toBeFlushedMu.RUnlock()
				continue
			}

			// Copy s.toBeFlushed to flushValues []SeqValue (local variable)
			for key, value := range s.toBeFlushed {
				flushValues = append(flushValues, SeqValue{
					Key:   key,
					Value: value,
				})
			}
		}
		s.toBeFlushedMu.RUnlock()

		// Error handling: Handle errors with retry mechanism (500ms wait)
		err := coreutils.Retry(s.cleanupCtx, s.iTime, func() error {
			return s.seqStorage.WriteValuesAndNextPLogOffset(flushValues, flushOffset)
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// the only case here - ctx is closed during storage error
				return
			}
			// notest
			panic("failed to flush values: " + err.Error())
		}

		// for each key in flushValues remove key from s.toBeFlushed if values are the same
		s.toBeFlushedMu.Lock()
		for _, fv := range flushValues {
			if v, exist := s.toBeFlushed[fv.Key]; exist && v == fv.Value {
				delete(s.toBeFlushed, fv.Key)
			}
		}
		s.toBeFlushedMu.Unlock()
	}
}

// Next implements isequencer.ISequencer.Next.
// It ensures thread-safe access to sequence values and handles various caching layers.
//
// Flow:
// - Validate equencing Transaction status
// - Get initialValue from s.params.SeqTypes and ensure that SeqID is known
// - Try to obtain the next value using:
//   - Try s.cache (can be evicted)
//   - Try s.inproc
//   - Try s.toBeFlushed (use s.toBeFlushedMu to synchronize)
//   - Try s.params.SeqStorage.ReadNumber()
//   - Read all known Numbers for wsKind, wsID
//   - If number is 0 then initial value is used
//   - Write all Numbers to s.cache
//
// - Write value+1 to s.cache
// - Write value+1 to s.inproc
// - Return value
// [~server.design.sequences/cmp.sequencer.Next~impl]
func (s *sequencer) Next(seqID SeqID) (num Number, err error) {
	// Validate sequencing Transaction status
	s.checkSequencingTransactionInProgress()

	// Get initialValue from s.params.SeqTypes and ensure that SeqID is known
	// existense is checked already on Start()
	seqTypes := s.params.SeqTypes[s.currentWSKind]

	initialValue, ok := seqTypes[seqID]
	if !ok {
		return 0, fmt.Errorf("%w: %d", ErrUnknownSeqID, seqID)
	}

	key := NumberKey{
		WSID:  s.currentWSID,
		SeqID: seqID,
	}

	// Try to obtain the next value using:
	// Try s.cache (can be evicted)
	if nextNumber, ok := s.cache.Get(key); ok {
		return s.incrementNumber(key, nextNumber), nil
	}

	// Try s.inproc
	lastNumber, ok := s.inproc[key]
	if ok {
		return s.incrementNumber(key, lastNumber), nil
	}

	// Try s.toBeFlushed (use s.toBeFlushedMu to synchronize)
	s.toBeFlushedMu.RLock()
	nextNumber, ok := s.toBeFlushed[key]
	s.toBeFlushedMu.RUnlock()
	if ok {
		return s.incrementNumber(key, nextNumber), nil
	}

	// Try s.params.SeqStorage.ReadNumber()
	var storedNumbers []Number
	err = coreutils.Retry(s.cleanupCtx, s.iTime, func() error {
		var err error
		// Read all known Numbers for wsKind, wsID
		storedNumbers, err = s.seqStorage.ReadNumbers(s.currentWSID, []SeqID{seqID})
		// Write all Numbers to s.cache
		for _, number := range storedNumbers {
			if number == 0 {
				continue
			}
			s.cache.Add(key, number)
		}
		return err
	})
	if err != nil {
		// happens when ctx is closed during storage error
		return 0, err
	}

	// If number is 0 then initial value is used
	nextNumber = storedNumbers[0]
	if nextNumber == 0 {
		nextNumber = initialValue - 1 // initial value 1 and there are no such records in plog at all -> should issue 1, not 2
	}

	// Write value+1 to s.cache
	// Write value+1 to s.inproc
	return s.incrementNumber(key, nextNumber), nil
}

// incrementNumber increments the number for the given key and returns the next number
func (s *sequencer) incrementNumber(key NumberKey, number Number) Number {
	nextNumber := number + 1
	// Write value+1 to s.cache
	s.cache.Add(key, nextNumber)
	// Write value+1 to s.inproc
	s.inproc[key] = nextNumber
	// Return value
	return nextNumber
}

// Flush implements isequencer.ISequencer.Flush.
// Flow:
//
//	Copy s.inproc and s.nextOffset to s.toBeFlushed and s.toBeFlushedOffset
//	Clear s.inproc
//	Increase s.nextOffset
//	Non-blocking write to s.flusherSig
//
// [~server.design.sequences/cmp.sequencer.Flush~impl]
func (s *sequencer) Flush() {
	// Verify processing is started
	s.checkSequencingTransactionInProgress()

	// wrong to skip if inproc is empty because need to flush new PLogOffset, see "flush offset without Next" test

	// Lock toBeFlushed while copying values
	s.toBeFlushedMu.Lock()
	// Copy s.inproc to s.toBeFlushed
	for key, value := range s.inproc {
		s.toBeFlushed[key] = value
	}

	// Copy s.nextOffset to s.toBeFlushedOffset
	s.toBeFlushedOffset = s.nextOffset
	s.toBeFlushedMu.Unlock()

	// Clear s.inproc
	s.inproc = make(map[NumberKey]Number)

	// Increase s.nextOffset
	s.nextOffset++
	//  Non-blocking write to s.flusherSig
	s.signalToFlushing()

	// Finish Sequencing Transaction
	s.finishSequencingTransaction()
}

func (s *sequencer) finishSequencingTransaction() {
	s.currentWSID = 0
	s.currentWSKind = 0
	s.transactionIsInProgress = false
}

// batcher processes a batch of sequence values and writes maximum values to storage.
// Flow:
// - Wait until len(s.toBeFlushed) < s.params.MaxNumUnflushedValues
//   - Lock/Unlock
//   - Sleep for s.params.BatcherDelayOnToBeFlushedOverflow
//   - check ctx (return ctx.Err())
//
// - s.nextOffset = offset + 1
// - Store maxValues in s.toBeFlushed: max Number for each SeqValue.Key
// - s.toBeFlushedOffset = offset + 1
func (s *sequencer) batcher(ctx context.Context, values []SeqValue, offset PLogOffset) error {
	// Wait until len(s.toBeFlushed) < s.params.MaxNumUnflushedValues
	for s.loadNumToBeFlushed() >= s.params.MaxNumUnflushedValues {
		s.signalToFlushing()
		delayCh := s.iTime.NewTimerChan(s.params.BatcherDelayOnToBeFlushedOverflow)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-delayCh:
		}
	}

	s.nextOffset = offset + 1
	s.toBeFlushedMu.Lock()
	s.toBeFlushedOffset = s.nextOffset
	s.toBeFlushedMu.Unlock()

	// Store maxValues in s.toBeFlushed: max Number for each SeqValue.Key
	maxValues := make(map[NumberKey]Number)
	for _, sv := range values {
		if current, exists := maxValues[sv.Key]; !exists || sv.Value > current {
			maxValues[sv.Key] = sv.Value
		}
	}

	s.toBeFlushedMu.Lock()
	maps.Copy(s.toBeFlushed, maxValues)
	s.toBeFlushedMu.Unlock()

	return nil
}

/*
actualizer is started in goroutine by Actualize().

Flow:

- if s.flusherWG is not nil
  - s.cancelFlusherCtx()
  - Wait for s.flusherWG
  - s.flusherWG = nil

- Clean s.toBeFlushed, toBeFlushedOffset
- s.flusherWG, s.flusherCtxCancel + start flusher() goroutine
- Read nextPLogOffset from s.params.SeqStorage.ReadNextPLogOffset()
- Use s.params.SeqStorage.ActualizeSequencesFromPLog() and s.batcher()
ctx handling:
  - if ctx is closed exit

Error handling:
- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) actualizer(actualizerCtx context.Context) {
	defer func() {
		// should be exactly that order, otherwise Start() could return false after New()
		// due of actualizerInProgress
		s.actualizerInProgress.Store(false)
		s.actualizerWG.Done()
	}()

	s.stopFlusher()

	// Clean s.toBeFlushed, toBeFlushedOffset
	// Locking is not necessary here as neither Start nor flusher is active
	s.toBeFlushed = map[NumberKey]Number{}
	s.toBeFlushedOffset = 0

	// s.flusherWG, s.flusherCtxCancel + start flusher() goroutine
	s.startFlusher()

	var err error
	// Read nextPLogOffset from s.params.SeqStorage.ReadNextPLogOffset()
	err = coreutils.Retry(actualizerCtx, s.iTime, func() error {
		s.nextOffset, err = s.seqStorage.ReadNextPLogOffset()
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// happens when ctx is closed during storage error
			return
		}
		// notest
		panic("failed to read last PLog offset: " + err.Error())
	}

	// Use s.params.SeqStorage.ActualizeSequencesFromPLog() and s.batcher()
	err = coreutils.Retry(actualizerCtx, s.iTime, func() error {
		return s.seqStorage.ActualizeSequencesFromPLog(actualizerCtx, s.nextOffset, s.batcher)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// happens when ctx is closed during storage error
			return
		}
		// notest
		panic("failed to actualize PLog: " + err.Error())
	}
}

// checkSequencingTransactionInProgress validates sequencing Transaction status
func (s *sequencer) checkSequencingTransactionInProgress() {
	if !s.transactionIsInProgress {
		panic("Sequencing Transaction is not in progress")
	}
}

// Actualize implements isequencer.ISequencer.Actualize.
// Flow:
// - Validate Actualization status
// - Validate Sequencing Transaction status
// - Set s.actualizerInProgress to true
// - Clean s.cache, s.nextOffset, s.currentWSID, s.currentWSKind, s.toBeFlushed, s.inproc, s.toBeFlushedOffset
// - Start the actualizer() goroutine
// [~server.design.sequences/cmp.sequencer.Actualize~impl]
func (s *sequencer) Actualize() {
	// Validate Sequencing Transaction status
	s.checkSequencingTransactionInProgress()

	// - Validate Actualization status
	// Set s.actualizerInProgress to true
	if !s.actualizerInProgress.CompareAndSwap(false, true) {
		panic("actualization is already in progress")
	}

	// Clean s.inproc
	s.inproc = make(map[NumberKey]Number)

	// do not clean toBeFlushed to avoid case:
	// Start Next Flush Start Next Actualize
	// Actualize() cleared toBeFlushed when previous flusher did not write it to storage yet
	// possible case in flusher: toBeFlushed is empty but plogoffset is not 0 -> Number does not match the PLogOffset

	// Cleans s.cache
	s.cache.Purge()

	// Cleans s.currentWSID, s.currentWSKind
	s.finishSequencingTransaction()

	// Start the actualizer() goroutine
	s.startActualizer()
}

// cleanup stops the actualizer() and flusher() goroutines.
func (s *sequencer) cleanup() {
	if s.actualizerInProgress.Load() {
		s.actualizerCtxCancel()
		s.actualizerWG.Wait()
		s.actualizerInProgress.Store(false)
	}
	s.stopFlusher()
	s.cleanupCtxCancel()
	s.finishSequencingTransaction()
}

func (s *sequencer) loadNumToBeFlushed() int {
	s.toBeFlushedMu.RLock()
	numToBeFlushed := len(s.toBeFlushed)
	s.toBeFlushedMu.RUnlock()
	return numToBeFlushed
}
