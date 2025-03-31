/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	"sync"

	"github.com/voedger/voedger/pkg/coreutils"
)

var (
	// variables for retry mechanism
	retryDelay = defaultRetryDelay
	retryCount = defaultRetryCount
)

// Start starts Sequencing Transaction for the given WSID.
// Marks Sequencing Transaction as in progress.
// Panics if Sequencing Transaction is already started.
// Normally returns the next PLogOffset, true
// Returns `0, false` if:
// - Actualization is in progress
// - The number of unflushed values exceeds the maximum threshold
// If ok is true, the caller must call Flush() or Actualize() to complete the Sequencing Transaction.
func (s *sequencer) Start(wsKind WSKind, wsID WSID) (plogOffset PLogOffset, ok bool) {
	// Check if cleanup is in progress
	if s.cleanupCtx.Err() != nil {
		panic("sequencer is in cleanup state")
	}

	// Check if Actualization is in progress
	if s.actualizerInProgress.Load() {
		return 0, false
	}

	// Panics if Sequencing Transaction is already started.
	if s.currentWSID != 0 || s.currentWSKind != 0 {
		panic("event processing is already started")
	}

	// Verify wsKind exists in supported types
	if _, exists := s.params.SeqTypes[wsKind]; !exists {
		panic("unknown wsKind")
	}

	// Check unflushed values threshold
	s.inprocMu.RLock()
	if len(s.inproc) >= s.params.MaxNumUnflushedValues {
		// The number of unflushed values exceeds the maximum threshold
		s.inprocMu.RUnlock()
		return 0, false
	}
	s.inprocMu.RUnlock()

	// Marks Sequencing Transaction as in progress.
	if !s.transactionIsInProgress.CompareAndSwap(false, true) {
		// notest
		panic("unexpected transaction in progress")
	}
	s.currentWSID = wsID
	s.currentWSKind = wsKind

	return s.nextOffset, true
}

func (s *sequencer) startFlusher() {
	if !s.flusherInProgress.CompareAndSwap(false, true) {
		// notest
		panic("flusher already started")
	}

	flusherCtx, flusherCtxCancel := context.WithCancel(context.Background())
	s.flusherCtxCancel = flusherCtxCancel
	s.flusherWG = &sync.WaitGroup{}
	s.flusherWG.Add(1)
	go s.flusher(flusherCtx)
}

func (s *sequencer) startActualizer() {
	// Check if actualization is in progress
	if !s.actualizerInProgress.CompareAndSwap(false, true) {
		// notest
		panic("unexpected actualization in progress")
	}

	actualizerCtx, actualizerCtxCancel := context.WithCancel(context.Background())
	s.actualizerCtxCancel = actualizerCtxCancel

	s.actualizerWG.Add(1)
	go s.actualizer(actualizerCtx)
}

func (s *sequencer) stopFlusher() {
	if s.flusherWG != nil {
		s.flusherCtxCancel()
		s.flusherWG.Wait()
		s.flusherWG = nil
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
- s.params.SeqStorage.WriteValuesAndOffset(flushValues, flushOffset)
- Lock s.toBeFlushedMu
- for each key in flushValues remove key from s.toBeFlushed if values are the same
- Unlock s.toBeFlushedMu

Error handling:

- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) flusher(ctx context.Context) {
	defer func() {
		s.flusherInProgress.Store(false)
		s.flusherWG.Done()
	}()

	// Non-blocking write to flusherStartedCh for tests
	select {
	case s.flusherStartedCh <- struct{}{}:
	default:
	}
	// Wait for ctx.Done()
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		// Wait for s.flusherSig
		case <-s.flusherSig:
		}
		s.toBeFlushedMu.Lock()
		flushOffset := s.toBeFlushedOffset
		s.toBeFlushedMu.Unlock()
		if err := s.flushValues(flushOffset, false); err != nil {
			// notest
			panic("failed to flush values: " + err.Error())
		}
	}
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
//   - Read all known Numbers for wsKind, wsID
//   - If number is 0 then initial value is used
//   - Write all Numbers to s.lru
//
// - Write value+1 to s.lru
// - Write value+1 to s.inproc
// - Return value
func (s *sequencer) Next(seqID SeqID) (num Number, err error) {
	// Validate sequencing Transaction status
	s.checkEventState()

	// Get initialValue from s.params.SeqTypes and ensure that SeqID is known
	seqTypes, exists := s.params.SeqTypes[s.currentWSKind]
	if !exists {
		// notest
		panic("unknown wsKind")
	}

	initialValue, ok := seqTypes[seqID]
	if !ok {
		panic("unknown seqID")
	}

	key := NumberKey{
		WSID:  s.currentWSID,
		SeqID: seqID,
	}
	// Try to obtain the next value using:
	// Try s.lru (can be evicted)
	if nextNumber, ok := s.lru.Get(key); ok {
		return s.incrementNumber(key, nextNumber), nil
	}

	// Try s.inproc
	s.inprocMu.RLock()
	lastNumber, ok := s.inproc[key]
	s.inprocMu.RUnlock()
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
	var knownNumbers []Number
	err = coreutils.Retry(s.cleanupCtx, s.iTime, retryDelay, retryCount, func() error {
		var err error
		// Read all known Numbers for wsKind, wsID
		knownNumbers, err = s.params.SeqStorage.ReadNumbers(s.currentWSID, []SeqID{seqID})
		// Write all Numbers to s.lru
		for _, number := range knownNumbers {
			if number == 0 {
				continue
			}
			s.lru.Add(key, number)
		}

		return err
	})
	if err != nil {
		return 0, err
	}

	// If number is 0 then initial value is used
	nextNumber = knownNumbers[0]
	if nextNumber == 0 {
		nextNumber = initialValue
	}

	// Write value+1 to s.lru
	// Write value+1 to s.inproc
	return s.incrementNumber(key, nextNumber), nil
}

// incrementNumber increments the number for the given key and returns the next number
func (s *sequencer) incrementNumber(key NumberKey, number Number) Number {
	s.inprocMu.Lock()
	defer s.inprocMu.Unlock()

	nextNumber := number + 1
	// Write value+1 to s.lru
	s.lru.Add(key, nextNumber)
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
func (s *sequencer) Flush() {
	// Verify processing is started
	s.checkEventState()

	// Skip if no values to flush
	s.inprocMu.RLock()
	if len(s.inproc) == 0 {
		s.inprocMu.RUnlock()
		s.finishSequencingTransaction()

		return
	}
	s.inprocMu.RUnlock()

	// Lock toBeFlushed while copying values
	s.toBeFlushedMu.Lock()
	// Copy s.inproc to s.toBeFlushed
	s.inprocMu.RLock()
	for key, value := range s.inproc {
		s.toBeFlushed[key] = value
	}
	s.inprocMu.RUnlock()

	// Copy s.nextOffset to s.toBeFlushedOffset
	s.toBeFlushedOffset = s.nextOffset
	s.toBeFlushedMu.Unlock()

	// Clear s.inproc
	s.inprocMu.Lock()
	s.inproc = make(map[NumberKey]Number)
	s.inprocMu.Unlock()

	// Increase s.nextOffset
	s.nextOffset++
	//  Non-blocking write to s.flusherSig
	select {
	case s.flusherSig <- struct{}{}:
		// notest
	default:
		// notest
	}

	// Finish Sequencing Transaction
	s.finishSequencingTransaction()
}

func (s *sequencer) finishSequencingTransaction() {
	s.transactionIsInProgress.Store(false)
	s.currentWSID = 0
	s.currentWSKind = 0
}

// batcher processes a batch of sequence values and writes maximum values to storage.
// Flow:
// - Wait until len(s.toBeFlushed) < s.params.MaxNumUnflushedValues
//   - Lock/Unlock
//   - Wait s.params.BatcherDelay
//   - check ctx (return ctx.Err())
//
// - s.nextOffset = offset + 1
// - Store maxValues in s.toBeFlushed: max Number for each SeqValue.Key
// - s.toBeFlushedOffset = offset + 1
func (s *sequencer) batcher(ctx context.Context, values []SeqValue, offset PLogOffset) error {
	// Wait until len(s.toBeFlushed) < s.params.MaxNumUnflushedValues
	s.toBeFlushedMu.RLock()
	for len(s.toBeFlushed) >= s.params.MaxNumUnflushedValues {
		s.toBeFlushedMu.RUnlock()

		delayCh := s.iTime.NewTimerChan(s.params.BatcherDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		// notest
		case <-delayCh:
		}

		// notest
		s.toBeFlushedMu.RLock()
	}
	s.toBeFlushedMu.RUnlock()

	s.toBeFlushedOffset = offset + 1
	s.nextOffset = offset + 1

	// Store maxValues in s.toBeFlushed: max Number for each SeqValue.Key
	maxValues := make(map[NumberKey]Number)
	for _, sv := range values {
		if current, exists := maxValues[sv.Key]; !exists || sv.Value > current {
			maxValues[sv.Key] = sv.Value
		}
	}

	s.toBeFlushedMu.Lock()
	for key, maxValue := range maxValues {
		s.toBeFlushed[key] = maxValue
	}
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
		s.actualizerWG.Done()
		s.actualizerInProgress.Store(false)
	}()

	// if s.flusherWG is not nil
	if s.flusherWG != nil {
		s.stopFlusher()
	}

	// Clean s.toBeFlushed, toBeFlushedOffset
	s.toBeFlushedMu.Lock()
	s.toBeFlushed = make(map[NumberKey]Number)
	s.toBeFlushedOffset = 0
	s.toBeFlushedMu.Unlock()

	// s.flusherWG, s.flusherCtxCancel + start flusher() goroutine
	s.startFlusher()

	var err error
	// Read nextPLogOffset from s.params.SeqStorage.ReadNextPLogOffset()
	err = coreutils.Retry(actualizerCtx, s.iTime, retryDelay, retryCount, func() error {
		s.nextOffset, err = s.params.SeqStorage.ReadNextPLogOffset()

		return err
	})
	if err != nil {
		// notest
		panic("failed to read last PLog offset: " + err.Error())
	}

	// Use s.params.SeqStorage.ActualizeSequencesFromPLog() and s.batcher()
	err = coreutils.Retry(actualizerCtx, s.iTime, retryDelay, retryCount, func() error {
		return s.params.SeqStorage.ActualizeSequencesFromPLog(s.cleanupCtx, s.nextOffset, s.batcher)
	})
	if err != nil {
		// notest
		panic("failed to actualize PLog: " + err.Error())
	}
}

// flushValues writes the accumulated sequence values to the storage.
// Flow:
// - Lock s.toBeFlushedMu
// - Copy s.toBeFlushedOffset to flushOffset (local variable)
// - Copy s.toBeFlushed to flushValues []SeqValue (local variable)
// - Unlock s.toBeFlushedMu
// - s.params.SeqStorage.WriteValuesAndOffset(flushValues, flushOffset)
// - Lock s.toBeFlushedMu
// - Clean s.toBeFlushed
// - Unlock s.toBeFlushedMu
// Parameters:
// - offset - PLogOffset to be written
// - needToCleanToBeFlushed - if true, clears s.toBeFlushed after writing values. Otherwise, only removes values that were written.
func (s *sequencer) flushValues(offset PLogOffset, needToCleanToBeFlushed bool) error {
	// Copy s.toBeFlushed to flushValues []SeqValue (local variable)
	s.toBeFlushedMu.RLock()
	flushValues := make([]SeqValue, 0, len(s.toBeFlushed))
	for key, value := range s.toBeFlushed {
		flushValues = append(flushValues, SeqValue{
			Key:   key,
			Value: value,
		})
	}
	s.toBeFlushedMu.RUnlock()

	// s.params.SeqStorage.WriteValuesAndOffset(flushValues, offset)
	// Error handling: Handle errors with retry mechanism (500ms wait)
	err := coreutils.Retry(s.cleanupCtx, s.iTime, retryDelay, retryCount, func() error {
		return s.params.SeqStorage.WriteValuesAndOffset(flushValues, offset)
	})
	if err != nil {
		return err
	}

	// for each key in flushValues remove key from s.toBeFlushed if values are the same
	switch needToCleanToBeFlushed {
	case true:
		s.toBeFlushedMu.Lock()
		s.toBeFlushed = make(map[NumberKey]Number)
		s.toBeFlushedMu.Unlock()
	default:
		s.toBeFlushedMu.Lock()
		for _, fv := range flushValues {
			if v, exist := s.toBeFlushed[fv.Key]; exist && v == fv.Value {
				delete(s.toBeFlushed, fv.Key)
			}
		}
		s.toBeFlushedMu.Unlock()
	}

	return nil
}

// checkEventState validates sequencing Transaction status
func (s *sequencer) checkEventState() {
	if s.currentWSID == 0 || s.currentWSKind == 0 {
		panic("event processing is not started")
	}
}

// Actualize implements isequencer.ISequencer.Actualize.
// Flow:
// - Validate Sequencing Transaction status (s.currentWSID != 0)
// - Validate Actualization status (s.actualizerInProgress is false)
// - Set s.actualizerInProgress to true
// - Clean s.lru, s.nextOffset, s.currentWSID, s.currentWSKind, s.toBeFlushed, s.inproc, s.toBeFlushedOffset
// - Start the actualizer() goroutine
func (s *sequencer) Actualize() {
	// Validate Sequencing Transaction status (s.currentWSID != 0)
	s.checkEventState()

	// Clean s.inproc
	s.inprocMu.Lock()
	if len(s.inproc) > 0 {
		s.inproc = make(map[NumberKey]Number)
	}
	s.inprocMu.Unlock()

	// Cleans s.toBeFlushed
	s.toBeFlushedMu.Lock()
	if len(s.toBeFlushed) > 0 {
		s.toBeFlushed = make(map[NumberKey]Number)
	}
	s.toBeFlushedMu.Unlock()

	// Cleans s.toBeFlushedOffset
	s.toBeFlushedOffset = 0
	// Cleans s.lru
	s.lru.Purge()

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

	if s.flusherWG != nil {
		s.flusherCtxCancel()
		s.flusherWG.Wait()
	}

	s.cleanupCtxCancel()
}
