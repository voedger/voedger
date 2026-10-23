/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package commandprocessor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecoveryManager_BasicUsage(t *testing.T) {
	require := require.New(t)
	manager := newTestRecoveryManager(t, nil)
	blockRecovering := make(chan struct{})
	factoryCalls := 0

	expectedReoveredValue := &valueToRecover{id: 1}

	recoveryKey := 10

	// first get -> the recovery is started but not yet done -> errRecoveryInProgress
	actualRecoveredValue, err := manager.getOrStart(t.Context(), recoveryKey,
		func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return func(context.Context) (*valueToRecover, error) {
				<-blockRecovering
				return expectedReoveredValue, nil
			}
		})
	require.Nil(actualRecoveredValue)
	require.ErrorIs(err, errRecoveryInProgress)
	manager.requireScheduled(recoveryKey)
	manager.requireStarted(recoveryKey)

	// try to query again for the same key -> still -> errRecoveryInProgress but factory not started again
	// because this key is being recovering now
	actualRecoveredValue, err = manager.getOrStart(t.Context(), recoveryKey,
		func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return nil
		})
	require.Nil(actualRecoveredValue)
	require.ErrorIs(err, errRecoveryInProgress)
	require.Equal(1, factoryCalls) // not started 2nd time

	// let the recovery finish
	close(blockRecovering)
	require.NoError(manager.requireCompleted(recoveryKey))

	// successfully recovered -> getOrStart returns the recovered value
	actualRecoveredValue, err = manager.getOrStart(t.Context(), 10,
		func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return nil
		})
	require.NoError(err)
	require.Same(expectedReoveredValue, actualRecoveredValue)
	require.Equal(1, factoryCalls)
	manager.shutdown()
}

type valueToRecover struct {
	id int
}

type recoveryResult struct {
	key int
	err error
}

type testRecoveryManager struct {
	*recoveryManager[int, valueToRecover]
	t           *testing.T
	scheduledCh chan int
	startedCh   chan int
	completedCh chan recoveryResult
}

func newTestRecoveryManager(t *testing.T, slots chan struct{}) *testRecoveryManager {
	t.Helper()
	manager := &testRecoveryManager{
		t:           t,
		scheduledCh: make(chan int, 10),
		startedCh:   make(chan int, 10),
		completedCh: make(chan recoveryResult, 10),
	}
	manager.recoveryManager = newRecoveryManager(slots, recoveryHooks[int, valueToRecover]{
		scheduled: func(key int) {
			manager.scheduledCh <- key
		},
		attempt: func(key int, next recoveryAttemptFunc[valueToRecover]) recoveryAttemptFunc[valueToRecover] {
			return func(ctx context.Context) (*valueToRecover, error) {
				manager.startedCh <- key
				return next(ctx)
			}
		},
		attemptCompleted: func(key int, err error) {
			manager.completedCh <- recoveryResult{key: key, err: err}
		},
	})
	return manager
}

func (m *testRecoveryManager) requireScheduled(key int) {
	m.t.Helper()
	require.Equal(m.t, key, timedReadChan(m.t, m.scheduledCh))
}

func (m *testRecoveryManager) requireStarted(key int) {
	m.t.Helper()
	require.Equal(m.t, key, timedReadChan(m.t, m.startedCh))
}

func (m *testRecoveryManager) requireCompleted(key int) error {
	m.t.Helper()
	completion := timedReadChan(m.t, m.completedCh)
	require.Equal(m.t, key, completion.key)
	return completion.err
}

func timedReadChan[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for recover manager")
		var zero T
		return zero
	}
}

func TestRecoveryManager(t *testing.T) {
	t.Run("retains a failure and starts exactly one retry", func(t *testing.T) {
		require := require.New(t)
		manager := newTestRecoveryManager(t, nil)
		recoveryErr := errors.New("recovery failed")
		recoveryKey := 1
		retryGate := make(chan struct{})
		retryStarted := make(chan struct{})
		expected := &valueToRecover{id: 2}
		attemptCalls := 0
		recoveryAttempt := func(context.Context) (*valueToRecover, error) {
			attemptCalls++
			switch attemptCalls {
			case 1:
				return nil, recoveryErr
			case 2:
				close(retryStarted)
				<-retryGate
				return expected, nil
			default:
				return nil, errors.New("unexpected recovery attempt")
			}
		}
		factoryCalls := 0
		newAttempt := func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return recoveryAttempt
		}

		// 1st query -> errRecoveryInProgress
		// but then the recovery is completed with an error
		recovered, err := manager.getOrStart(t.Context(), recoveryKey, newAttempt)
		require.Nil(recovered)
		require.ErrorIs(err, errRecoveryInProgress)
		require.ErrorIs(manager.requireCompleted(recoveryKey), recoveryErr)

		// 2nd query -> recovery error is returned
		// then next recovery attempt is started
		recovered, err = manager.getOrStart(t.Context(), recoveryKey, newAttempt)
		require.Nil(recovered)
		require.ErrorIs(err, recoveryErr)
		timedReadChan(t, retryStarted)

		// 3rd query -> still waiting for 2nd attempt to finish, it is blocked -> errRecoveryInProgress
		recovered, err = manager.getOrStart(t.Context(), recoveryKey, newAttempt)
		require.Nil(recovered)
		require.ErrorIs(err, errRecoveryInProgress)
		require.Equal(2, factoryCalls)
		require.Equal(2, attemptCalls)

		// let 2nd attempt to successfully finish
		close(retryGate)
		require.NoError(manager.requireCompleted(recoveryKey))
		recovered, err = manager.getOrStart(t.Context(), recoveryKey, nil)
		require.NoError(err)
		require.Same(expected, recovered)
		manager.shutdown()
	})

	t.Run("does not create an attempt when the recovery limit is reached", func(t *testing.T) {
		require := require.New(t)
		// Recovery Manager with 1 recovery slot
		manager := newTestRecoveryManager(t, make(chan struct{}, 1))
		blockAttempt := make(chan struct{})
		expected := &valueToRecover{id: 1}
		attemptStarted := make(chan struct{})
		attemptCalls := 0
		recoveryAttempt := func(context.Context) (*valueToRecover, error) {
			attemptCalls++
			close(attemptStarted)
			<-blockAttempt
			return expected, nil
		}
		factoryCalls := 0
		newAttempt := func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return recoveryAttempt
		}

		// 1st query -> errRecoveryInProgress, recovery ettempt is blocked
		_, err := manager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryInProgress)
		manager.requireStarted(1)
		timedReadChan(t, attemptStarted)

		// 2nd query another key, no slots remainnig -> errRecoveryLimit
		recovered, err := manager.getOrStart(t.Context(), 2, newAttempt)
		require.Nil(recovered)
		require.ErrorIs(err, errRecoveryLimit)
		require.Equal(1, factoryCalls)
		require.Equal(1, attemptCalls)

		// let the 1st attempt to finish
		close(blockAttempt)
		require.NoError(manager.requireCompleted(1))
		manager.shutdown()

		// Reocvery Manager with no ability to recover
		zeroLimitManager := newTestRecoveryManager(t, make(chan struct{}))
		_, err = zeroLimitManager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryLimit)
		require.Equal(1, factoryCalls)
		require.Equal(1, attemptCalls)
		zeroLimitManager.shutdown()
	})

	t.Run("a failed value cannot retry until a recovery slot is available", func(t *testing.T) {
		require := require.New(t)
		manager := newTestRecoveryManager(t, make(chan struct{}, 1))
		errRecoveryFailure := errors.New("first attempt failed")
		blockAttempt2 := make(chan struct{})
		attempt2Started := make(chan struct{})
		retryValue := &valueToRecover{id: 3}
		attemptCalls := 0
		recoveryAttempt := func(context.Context) (*valueToRecover, error) {
			attemptCalls++
			switch attemptCalls {
			case 1:
				return nil, errRecoveryFailure
			case 2:
				close(attempt2Started)
				<-blockAttempt2
				return &valueToRecover{id: 2}, nil
			case 3:
				return retryValue, nil
			default:
				return nil, errors.New("unexpected recovery attempt")
			}
		}
		factoryCalls := 0
		newAttempt := func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return recoveryAttempt
		}

		// 1st query key 1-> errRecoveryInProgress, then completed with errRecoveryFailure
		_, err := manager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryInProgress)
		manager.requireStarted(1)
		require.ErrorIs(manager.requireCompleted(1), errRecoveryFailure)

		// 1st query key 2 -> errRecoveryInProgress but attempt is blocked
		_, err = manager.getOrStart(t.Context(), 2, newAttempt)
		require.ErrorIs(err, errRecoveryInProgress)
		manager.requireStarted(2)
		timedReadChan(t, attempt2Started)

		// 2nd query key 1 again -> errRecoveryLimit because the single slot is busy on attempt on key 2
		_, err = manager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryLimit)
		require.Equal(2, factoryCalls)
		require.Equal(2, attemptCalls)

		// let the attempt on key 2 finish
		close(blockAttempt2)
		require.NoError(manager.requireCompleted(2))

		// 3rd query key 1 -> errRecoveryFailure because the recovery is started
		_, err = manager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryFailure)
		require.NoError(manager.requireCompleted(1))
		require.Equal(3, factoryCalls)
		require.Equal(3, attemptCalls)

		// after key 1 recovered it is ok to query it
		recovered, err := manager.getOrStart(t.Context(), 1, nil)
		require.NoError(err)
		require.Same(retryValue, recovered)
		manager.shutdown()
	})

	t.Run("passes context to the attempt without interpreting it", func(t *testing.T) {
		require := require.New(t)
		manager := newTestRecoveryManager(t, nil)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		newAttempt := func() recoveryAttemptFunc[valueToRecover] {
			return func(ctx context.Context) (*valueToRecover, error) {
				return nil, ctx.Err()
			}
		}

		recovered, err := manager.getOrStart(ctx, 1, newAttempt)
		require.Nil(recovered)
		require.ErrorIs(err, errRecoveryInProgress)
		require.ErrorIs(manager.requireCompleted(1), context.Canceled)
		manager.shutdown()
	})

	t.Run("a stale completion cannot overwrite state created after reset", func(t *testing.T) {
		require := require.New(t)
		manager := newTestRecoveryManager(t, nil)
		attempt1Block := make(chan struct{})
		attempt2Block := make(chan struct{})
		attempt1RecoveredValue := &valueToRecover{id: 1}
		attempt2RecoveredValue := &valueToRecover{id: 2}
		attemptStarted := make(chan int, 2)
		var attemptCalls atomic.Int32
		recoveryAttempt := func(context.Context) (*valueToRecover, error) {
			attemptCall := int(attemptCalls.Add(1))
			attemptStarted <- attemptCall
			switch attemptCall {
			case 1:
				<-attempt1Block
				return attempt1RecoveredValue, nil
			case 2:
				<-attempt2Block
				return attempt2RecoveredValue, nil
			default:
				return nil, errors.New("unexpected recovery attempt")
			}
		}
		factoryCalls := 0
		newAttempt := func() recoveryAttemptFunc[valueToRecover] {
			factoryCalls++
			return recoveryAttempt
		}
		var attempt1Unblock sync.Once
		var attempt2Unblock sync.Once
		defer func() {
			attempt1Unblock.Do(func() { close(attempt1Block) })
			attempt2Unblock.Do(func() { close(attempt2Block) })
			manager.shutdown()
		}()

		// 1st query -> errRecoveryInProgress; the attempt starts and remains blocked on attempt1Block
		_, err := manager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryInProgress)
		manager.requireStarted(1)
		require.Equal(1, timedReadChan(t, attemptStarted))

		// Reset key 1 while its old attempt is still in progress
		manager.reset(1)

		// 2nd query for key 1 -> errRecoveryInProgress; a new attempt starts and remains blocked on attempt2Block
		_, err = manager.getOrStart(t.Context(), 1, newAttempt)
		require.ErrorIs(err, errRecoveryInProgress)
		manager.requireStarted(1)
		require.Equal(2, timedReadChan(t, attemptStarted))

		// Let the attempt 1 finish; its stale completion must not replace or clear the new in-progress state
		attempt1Unblock.Do(func() { close(attempt1Block) })

		// wait for attempt1 completed
		require.NoError(manager.requireCompleted(1))

		// still in progress despite attempt1 is completed
		// because key1 leads to attempt2 inside the recoveryManager, attempt1 is stale
		recovered, err := manager.getOrStart(t.Context(), 1, newAttempt)
		require.Nil(recovered)
		require.ErrorIs(err, errRecoveryInProgress)
		require.Equal(2, factoryCalls)

		// Let the new attempt finish; key 1 must expose the new value without starting another attempt
		attempt2Unblock.Do(func() { close(attempt2Block) })

		// wait for 2nd attempt complete
		require.NoError(manager.requireCompleted(1))

		// now check that the actual value is value recovered on attempt2, not on the outdated attempt1
		recovered, err = manager.getOrStart(t.Context(), 1, newAttempt)
		require.NoError(err)
		require.Same(attempt2RecoveredValue, recovered)
		require.Equal(2, factoryCalls)
		require.Equal(int32(2), attemptCalls.Load())
	})
}
