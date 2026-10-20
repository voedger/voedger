/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"context"
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/istructs"
)

// recoveryManager coordinates asynchronous recovery when many requests can ask
// for the same key at once and recovery capacity is bounded:
//
//	request A --\
//	request B ---+--> values[key] --> one recovery goroutine --> ready *T
//	request C --/          |                 |
//	                       |                 +--> retained failure --> one retry
//	                       +--> recovering requests get "in progress"
//
//	new key --> reserve slot --> install state --> start goroutine
//	            |
//	            +-- no slot --> reject without creating an attempt
//
// The map deduplicates work per key, slots bound concurrent recoveries, and
// workers let shutdown wait for every goroutine.
type recoveryManager[K comparable, T any] struct {
	mu      sync.Mutex
	values  map[K]*recoverableValue[T]
	slots   chan struct{} // nil means unlimited recovery concurrency
	workers *sync.WaitGroup
	hooks   recoveryHooks[K, T]
}

// recoverableValue is the per-key state machine stored by recoveryManager:
//
//	value == nil, recoveryErr == nil  --> recovering
//	value == nil, recoveryErr != nil  --> failed; report error and start one retry
//	value != nil, recoveryErr == nil  --> ready; all requests share the value
//
// Absence from the map is the fourth state: recovery has not started.
type recoverableValue[T any] struct {
	value       *T
	recoveryErr error
}

type recoveryAttemptFunc[T any] func(context.Context) (*T, error)

// newRecoveryAttemptFunc creates an attempt only after recoveryManager has reserved a recovery slot.
// This lets the caller synchronously transfer request-owned resources to the asynchronous attempt;
// creating it eagerly could orphan those resources when no attempt is started.
type newRecoveryAttemptFunc[T any] func() recoveryAttemptFunc[T]

type recoveryHooks[K comparable, T any] struct {
	scheduled        func(K)
	attempt          func(K, recoveryAttemptFunc[T]) recoveryAttemptFunc[T]
	attemptCompleted func(K, error)
	onLogRead        func(K, istructs.Offset, int)
}

func newRecoveryManager[K comparable, T any](slots chan struct{}, hooks recoveryHooks[K, T]) *recoveryManager[K, T] {
	return &recoveryManager[K, T]{
		values:  map[K]*recoverableValue[T]{},
		slots:   slots,
		workers: &sync.WaitGroup{},
		hooks:   hooks,
	}
}

func (m *recoveryManager[K, T]) getOrStart(vvmCtx context.Context, key K,
	newAttempt newRecoveryAttemptFunc[T]) (*T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value := m.values[key]
	if value == nil {
		// recover was not started -> just start if max simultaneous recoveries limit is not reached
		if !m.reserveSlot() {
			return nil, errRecoveryLimit
		}
		value = &recoverableValue[T]{}
		m.values[key] = value
		// The slot is ours, so caller-owned resources can now be transferred to the attempt.
		m.startRecover(vvmCtx, key, value, newAttempt())
		return nil, errRecoveryInProgress
	}
	if value.value != nil {
		// recover is successfully finished
		return value.value, nil
	}
	if value.recoveryErr == nil {
		// no value and no error -> recovery is in progress
		return nil, errRecoveryInProgress
	}
	// no value and has error -> will restart recovery if max simultaneous recoveries limit is not reached
	if !m.reserveSlot() {
		return nil, errRecoveryLimit
	}
	previousErr := value.recoveryErr
	value.recoveryErr = nil
	// Create a fresh attempt for the retry only after its slot has been reserved.
	m.startRecover(vvmCtx, key, value, newAttempt())
	return nil, previousErr
}

func (m *recoveryManager[K, T]) reserveSlot() bool {
	if m.slots == nil {
		return true
	}
	select {
	case m.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (m *recoveryManager[K, T]) startRecover(vvmCtx context.Context, key K, value *recoverableValue[T], attempt recoveryAttemptFunc[T]) {
	m.workers.Add(1)
	m.hooks.scheduled(key)
	attempt = m.hooks.attempt(key, attempt)
	go m.recover(vvmCtx, key, value, attempt)
}

func (m *recoveryManager[K, T]) recover(vvmCtx context.Context, key K, value *recoverableValue[T], attempt recoveryAttemptFunc[T]) {
	defer m.workers.Done()
	recovered, err := attempt(vvmCtx)
	if err != nil {
		err = fmt.Errorf("%w: %w", errRecoveryFailed, err)
	}
	m.complete(value, recovered, err)
	m.hooks.attemptCompleted(key, err)
}

//	attempt result --> captured state --> ready or failed
//	               +------------------> release recovery slot
//
// A reset may detach the state from the map, but cannot make this pointer refer
// to a newer attempt's state.
func (m *recoveryManager[K, T]) complete(value *recoverableValue[T], recovered *T, err error) {
	m.mu.Lock()
	value.recoveryErr = err
	if err == nil {
		value.value = recovered
	} else {
		value.value = nil
	}
	m.mu.Unlock()
	if m.slots != nil {
		<-m.slots
	}
}

func (m *recoveryManager[K, T]) reset(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
}

func (m *recoveryManager[K, T]) clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values = map[K]*recoverableValue[T]{}
}

func (m *recoveryManager[K, T]) shutdown() {
	m.workers.Wait()
	m.clear()
}

func (m *recoveryManager[K, T]) recoveredValues() map[K]*T {
	m.mu.Lock()
	defer m.mu.Unlock()
	values := map[K]*T{}
	for key, value := range m.values {
		if value.value != nil {
			values[key] = value.value
		}
	}
	return values
}
