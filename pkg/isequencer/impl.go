/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package isequencer

import (
	"context"
	lruPkg "github.com/hashicorp/golang-lru/v2"

	"github.com/voedger/voedger/pkg/coreutils"
)

// New creates a new sequencer
func New(params *Params, iTime coreutils.ITime) (ISequencer, context.CancelFunc) {
	lru, err := lruPkg.New[NumberKey, Number](params.LRUCacheSize)
	if err != nil {
		panic("failed to create LRU cache: " + err.Error())
	}

	cleanupCtx, cleanupCtxCancel := context.WithCancel(context.Background())
	s := &sequencer{
		params:           params,
		lru:              lru,
		toBeFlushed:      make(map[NumberKey]Number),
		inproc:           make(map[NumberKey]Number),
		cleanupCtx:       cleanupCtx,
		cleanupCtxCancel: cleanupCtxCancel,
		iTime:            iTime,
		flusherStartedCh: make(chan struct{}, 1),
	}

	s.startFlusher()
	<-s.flusherStartedCh

	return s, s.cleanup
}

// checkCleanupState panics if cleanup is in progress
func (s *sequencer) checkCleanupState() {
	select {
	case <-s.cleanupCtx.Done():
		panic("sequencer is in cleanup state")
	default:
	}
}

func (s *sequencer) Start(wsKind WSKind, wsID WSID) (plogOffset PLogOffset, ok bool) {
	// Check if cleanup is in progress
	s.checkCleanupState()

	// Check if actualization is in progress
	if s.actualizerInProgress.Load() {
		return 0, false
	}

	// Check if current processing is already started
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
		s.inprocMu.RUnlock()
		return 0, false
	}
	s.inprocMu.RUnlock()

	// Read last offset
	lastOffset, err := s.params.SeqStorage.ReadLastWrittenPLogOffset()
	if err != nil {
		panic("failed to read last PLog offset: " + err.Error())
	}

	// Starts event processing for the given WSID.
	s.currentWSID = wsID
	s.currentWSKind = wsKind
	s.inprocOffset = lastOffset + 1

	return s.inprocOffset, true
}

func (s *sequencer) startFlusher() {
	if s.flusherCtx != nil {
		panic("flusher already started")
	}

	s.flusherCtx, s.flusherCtxCancel = context.WithCancel(context.Background())
	s.flusherWg.Add(1)
	go s.flusher()
}

// flusher runs in a goroutine to periodically flush values from toBeFlushed to storage
func (s *sequencer) flusher() {
	defer s.flusherWg.Done()
	tickerCh := s.iTime.NewTimerChan(s.params.MaxFlushingInterval)
	s.flusherStartedCh <- struct{}{}
	for {
		select {
		case <-s.cleanupCtx.Done():
			return // Stop flusher when cleanup is in progress
		case <-s.flusherCtx.Done():
			return // Stop flusher when flusher is cancelled
		case <-tickerCh:
			tickerCh = s.iTime.NewTimerChan(s.params.MaxFlushingInterval)
			// Lock toBeFlushed while creating a batch
			s.toBeFlushedMu.Lock()
			if len(s.toBeFlushed) == 0 {
				s.toBeFlushedMu.Unlock()
				continue
			}

			// Create batch from toBeFlushed
			batch := make([]SeqValue, 0, len(s.toBeFlushed))
			offset := s.toBeFlushedOffset

			for key, value := range s.toBeFlushed {
				batch = append(batch, SeqValue{
					Key:   key,
					Value: value,
				})
			}

			// Write batch to storage
			// Call Actualize on error
			err := coreutils.Retry(s.cleanupCtx, s.iTime, retryDelay, retryCount, func() error {
				return s.params.SeqStorage.WriteValues(batch)
			})
			if err != nil {
				s.Actualize()
			}

			// Clear toBeFlushed before unlocking
			s.toBeFlushed = make(map[NumberKey]Number)
			s.toBeFlushedMu.Unlock()

			// Write offset after successful values write
			// Call Actualize on error
			err = coreutils.Retry(s.cleanupCtx, s.iTime, retryDelay, retryCount, func() error {
				return s.params.SeqStorage.WritePLogOffset(offset)
			})
			if err != nil {
				s.Actualize()
			}
		}
	}
}

func (s *sequencer) Next(seqID SeqID) (num Number, err error) {
	// 1. Validate processing status
	s.checkEventState()

	// 2. Get initialValue and verify seqID exists
	seqTypes, exists := s.params.SeqTypes[s.currentWSKind]
	if !exists {
		panic("unknown wsKind")
	}
	initialValue, exists := seqTypes[seqID]
	if !exists {
		panic("unknown seqID")
	}

	key := NumberKey{
		WSID:  s.currentWSID,
		SeqID: seqID,
	}

	// Helper function to increment and store value
	incrementNumber := func(value Number) Number {
		s.inprocMu.Lock()
		defer s.inprocMu.Unlock()

		nextValue := value + 1
		s.lru.Add(key, nextValue)
		s.inproc[key] = nextValue
		return nextValue
	}

	// 3. Try to obtain next value using cache hierarchy
	if value, ok := s.lru.Get(key); ok {
		return incrementNumber(value), nil
	}

	s.inprocMu.RLock()
	if value, ok := s.inproc[key]; ok {
		s.inprocMu.RUnlock()
		return incrementNumber(value), nil
	}
	s.inprocMu.RUnlock()

	s.toBeFlushedMu.RLock()
	value, ok := s.toBeFlushed[key]
	s.toBeFlushedMu.RUnlock()
	if ok {
		return incrementNumber(value), nil
	}

	// Try storage as last resort
	numbers, err := s.params.SeqStorage.ReadNumbers(s.currentWSID, []SeqID{seqID})
	if err != nil {
		return 0, err
	}

	value = numbers[0]
	if value == 0 {
		value = initialValue
	}

	return incrementNumber(value), nil
}

func (s *sequencer) Flush() {
	// Verify processing is started
	s.checkEventState()

	// Skip if no values to flush

	s.inprocMu.RLock()
	if len(s.inproc) == 0 {
		s.inprocMu.RUnlock()
		s.finishEventState()
		return
	}
	s.inprocMu.RUnlock()

	// Lock toBeFlushed while copying values
	s.toBeFlushedMu.Lock()
	defer s.toBeFlushedMu.Unlock()

	// Copy inproc values to toBeFlushed
	s.inprocMu.RLock()
	for key, value := range s.inproc {
		s.toBeFlushed[key] = value
	}
	s.inprocMu.RUnlock()

	// Update toBeFlushedOffset
	s.toBeFlushedOffset = s.inprocOffset

	// Reset current state
	s.finishEventState()
}

func (s *sequencer) finishEventState() {
	s.inprocMu.Lock()
	defer s.inprocMu.Unlock()

	if len(s.inproc) > 0 {
		s.inproc = make(map[NumberKey]Number)
	}
	s.currentWSID = 0
	s.currentWSKind = 0
}

// Flow:
// - Build maxValues: max Number for each SeqValue.Key
// - Write maxValues using s.params.SeqStorage.WriteValues()
func (s *sequencer) batcher(batch []SeqValue, batchOffset PLogOffset) error {
	// Aggregate max values per key
	maxValues := make(map[NumberKey]Number)
	for _, sv := range batch {
		if current, exists := maxValues[sv.Key]; !exists || sv.Value > current {
			maxValues[sv.Key] = sv.Value
		}
	}

	// Convert maxValues to batch
	valueBatch := make([]SeqValue, 0, len(maxValues))
	for key, value := range maxValues {
		valueBatch = append(valueBatch, SeqValue{
			Key:   key,
			Value: value,
		})
	}

	// Write max values to storage
	if err := s.params.SeqStorage.WriteValues(valueBatch); err != nil {
		return err
	}

	// Update offset after successful write
	if err := s.params.SeqStorage.WritePLogOffset(batchOffset); err != nil {
		return err
	}

	return nil
}

func (s *sequencer) actualizer() {
	defer s.actualizerWg.Done()

	// Retry delay on errors
	// Get last written offset to start actualization from
	var (
		offset PLogOffset
		err    error
	)

	err = coreutils.Retry(s.cleanupCtx, s.iTime, retryDelay, retryCount, func() error {
		offset, err = s.params.SeqStorage.ReadLastWrittenPLogOffset()
		return err
	})
	if err != nil {
		return
	}

	for {
		select {
		case <-s.cleanupCtx.Done():
			return // Stop actualization when context is cancelled
		default:
			err := coreutils.Retry(s.cleanupCtx, s.iTime, retryDelay, retryCount, func() error {
				return s.params.SeqStorage.ActualizePLog(s.cleanupCtx, offset, s.batcher)
			})
			if err == nil {
				// Start flusher after successful actualization
				s.startFlusher()
				return
			}
		}
	}
}

func (s *sequencer) checkEventState() {
	if s.currentWSID == 0 || s.currentWSKind == 0 {
		panic("event processing is not started")
	}
}

func (s *sequencer) Actualize() {
	// Verify processing is started
	s.checkEventState()

	if s.actualizerInProgress.Load() {
		panic("actualization is already in progress")
	}

	// Check if cleanup process is in progress
	s.checkCleanupState()

	// Stop flusher if running
	if s.flusherCtx != nil {
		select {
		case <-s.flusherCtx.Done():
			// Flusher already stopped
		default:
			// Cancel flusher context and wait
			s.flusherCtxCancel()
			s.flusherWg.Wait()
			s.flusherCtx = nil
		}
	}

	// Copy current values to toBeFlushed
	s.toBeFlushedMu.Lock()

	s.inprocMu.RLock()
	for key, value := range s.inproc {
		s.toBeFlushed[key] = value
	}
	s.inprocMu.RUnlock()
	s.toBeFlushedOffset = s.inprocOffset

	s.toBeFlushedMu.Unlock()

	// Reset current state
	s.finishEventState()

	// Start actualization
	s.actualizerInProgress.Store(true)
	s.actualizerWg.Add(1)
	go s.actualizer()
}

func (s *sequencer) cleanup() {
	// Stop flusher if running
	if s.flusherCtx != nil {
		s.flusherCtxCancel()
		s.flusherWg.Wait()
	}

	s.cleanupCtxCancel()

	// Stop actualizer if running
	s.actualizerInProgress.Store(false)
	s.actualizerWg.Wait()
}
