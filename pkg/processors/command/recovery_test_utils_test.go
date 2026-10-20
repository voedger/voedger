/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istructs"
)

// logReadFromTo records a recovery log read start offset and count.
type logReadFromTo struct {
	offset istructs.Offset
	count  int
}

// logReadsRecorder keeps synchronized snapshots of reads per recovery key.
type logReadsRecorder[K comparable] struct {
	mu    sync.Mutex
	byKey map[K][]logReadFromTo
}

func newLogReadsRecorder[K comparable]() *logReadsRecorder[K] {
	return &logReadsRecorder[K]{byKey: map[K][]logReadFromTo{}}
}

func (r *logReadsRecorder[K]) record(key K, offset istructs.Offset, count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byKey[key] = append(r.byKey[key], logReadFromTo{offset: offset, count: count})
}

func (r *logReadsRecorder[K]) reset(key K) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byKey, key)
}

// values returns a copy so callers cannot mutate the recorded reads.
func (r *logReadsRecorder[K]) values(key K) []logReadFromTo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]logReadFromTo(nil), r.byKey[key]...)
}

// recoveryAttempt makes one asynchronous recovery deterministic for a test:
//
//	test closes gate --> recovery may run --> manager publishes err --> close(done)
//	        |                                                            |
//	        +-- leave open to assert "in progress"                       +--> wait returns
//
// gate is optional. err initially carries an injected failure and is replaced
// with the manager's final wrapped result when the attempt completes.
type recoveryAttempt struct {
	done chan struct{}
	gate <-chan struct{}
	err  error
}

// recoveryTestAttempts solves deterministic testing of asynchronous managers:
//
//	blockNext(key) ----> nextGates[key] ---\
//	failNext(key, err) -> nextFailures[key] +--> scheduled(key) --> attempts[key]
//	request --------------------------------/          |                 |
//	                                                   +--> starts[key]  +--> wrapAttempt
//	                                                                         |
//	                                                                         +--> wait(key)
//
// Pending controls are consumed by exactly one scheduled attempt. starts proves
// that duplicate and over-limit requests did not launch work. K lets the same
// mechanism control both partition and workspace recovery.
type recoveryTestAttempts[K comparable, T any] struct {
	mu           sync.Mutex
	attempts     map[K]*recoveryAttempt
	starts       map[K]int
	nextGates    map[K]<-chan struct{}
	nextFailures map[K]error
}

func newRecoveryTestAttempts[K comparable, T any]() *recoveryTestAttempts[K, T] {
	return &recoveryTestAttempts[K, T]{
		attempts:     map[K]*recoveryAttempt{},
		starts:       map[K]int{},
		nextGates:    map[K]<-chan struct{}{},
		nextFailures: map[K]error{},
	}
}

func (c *recoveryTestAttempts[K, T]) hooks() recoveryHooks[K, T] {
	return recoveryHooks[K, T]{
		scheduled:        c.scheduled,
		attempt:          c.wrapAttempt,
		attemptCompleted: c.attemptCompleted,
	}
}

// scheduled consumes controls only after recoveryManager admits the work:
//
//	next gate/failure --> recoveryAttempt --> attempts[key]
//	                                      \-> starts[key]++
//
// Requests rejected by the concurrency limit never reach this method.
func (c *recoveryTestAttempts[K, T]) scheduled(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.starts[key]++
	c.attempts[key] = &recoveryAttempt{
		done: make(chan struct{}),
		gate: c.nextGates[key],
		err:  c.nextFailures[key],
	}
	delete(c.nextGates, key)
	delete(c.nextFailures, key)
}

// wrapAttempt turns test controls into recovery behavior inside the admitted
// recovery attempt:
//
//	gate open   --> attempt(ctx) --> wait ----> gate closed --> injected error or production attempt
//	ctx canceled --------------------------^---------------> context error
//
// Context participation lets service shutdown release a deliberately blocked
// attempt.
func (c *recoveryTestAttempts[K, T]) wrapAttempt(key K, next recoveryAttemptFunc[T]) recoveryAttemptFunc[T] {
	c.mu.Lock()
	attempt := c.attempts[key]
	c.mu.Unlock()
	return func(ctx context.Context) (*T, error) {
		if attempt.gate != nil {
			select {
			case <-attempt.gate:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if attempt.err != nil {
			return nil, attempt.err
		}
		return next(ctx)
	}
}

func (c *recoveryTestAttempts[K, T]) attemptCompleted(key K, err error) {
	c.mu.Lock()
	attempt := c.attempts[key]
	attempt.err = err
	close(attempt.done)
	c.mu.Unlock()
}

func (c *recoveryTestAttempts[K, T]) blockNext(key K) chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	gate := make(chan struct{})
	c.nextGates[key] = gate
	return gate
}

func (c *recoveryTestAttempts[K, T]) failNext(key K, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextFailures[key] = err
}

func (c *recoveryTestAttempts[K, T]) latest(key K) (*recoveryAttempt, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	attempt, ok := c.attempts[key]
	return attempt, ok
}

func (c *recoveryTestAttempts[K, T]) wait(ctx context.Context, key K) error {
	attempt, ok := c.latest(key)
	if !ok {
		return fmt.Errorf("recovery for %v was not started", key)
	}
	return attempt.wait(ctx)
}

func (a *recoveryAttempt) wait(ctx context.Context) error {
	select {
	case <-a.done:
		return a.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *recoveryTestAttempts[K, T]) startCount(key K) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.starts[key]
}

// recoveryTestControl wires the four independent recovery test channels:
//
//	partition recoveryManager --> partitions   command processor --> pLogReads
//	workspace recoveryManager --> workspaces   command processor --> wLogReads
//
// Keeping attempt control separate from read observation lets tests combine
// only the signals they need without leaking workspace state into partition
// assertions.
type recoveryTestControl struct {
	partitions *recoveryTestAttempts[partitionKey, appPartition]
	workspaces *recoveryTestAttempts[workspaceKey, workspace]
	pLogReads  *logReadsRecorder[partitionKey]
	wLogReads  *logReadsRecorder[workspaceKey]
}

func newRecoveryTestControl() *recoveryTestControl {
	return &recoveryTestControl{
		partitions: newRecoveryTestAttempts[partitionKey, appPartition](),
		workspaces: newRecoveryTestAttempts[workspaceKey, workspace](),
		pLogReads:  newLogReadsRecorder[partitionKey](),
		wLogReads:  newLogReadsRecorder[workspaceKey](),
	}
}

// testHooks connects the aggregate control to both recovery managers and the
// command processor:
//
//	partitions.hooks() --\
//	workspaces.hooks() ---+--> test fixture injection
//	pLogReads.record -----+
//	wLogReads.record -----/
func (c *recoveryTestControl) testHooks() (
	partitionHooks recoveryHooks[partitionKey, appPartition],
	workspaceHooks recoveryHooks[workspaceKey, workspace],
) {
	partitionHooks = c.partitions.hooks()
	partitionHooks.onLogRead = c.pLogReads.record
	workspaceHooks = c.workspaces.hooks()
	workspaceHooks.onLogRead = c.wLogReads.record
	return partitionHooks, workspaceHooks
}

// recoveryRetrySender waits for known recovery attempts in tests unrelated to recovery.
// Tests asserting admission responses use rawRequestSender instead.
type recoveryRetrySender struct {
	raw           bus.IRequestSender
	control       *recoveryTestControl
	keyForRequest func(bus.Request) (partitionKey, bool)
}

// SendRequest hides lazy partition and workspace recovery from unrelated tests:
//
//	request --> raw.SendRequest
//	              |
//	              +--> non-503/error/untracked request ----------> return response
//	              |
//	              +--> 503 + matching recovery message and attempt
//	                         |
//	                         +--> drain response --> wait(done) --> retry request
//	                                                  |
//	                                                  +--> failure/cancel --> return error
//
// Draining before waiting preserves the bus response lifecycle. Responses that
// do not identify a tracked recovery attempt are replayed unchanged to the caller.
func (s *recoveryRetrySender) SendRequest(ctx context.Context, req bus.Request) (<-chan any, bus.ResponseMeta, *error, error) {
	for {
		responseCh, responseMeta, responseErr, err := s.raw.SendRequest(ctx, req)
		key, retryRecovery := s.keyForRequest(req)
		if err != nil || responseMeta.StatusCode != http.StatusServiceUnavailable || !retryRecovery {
			return responseCh, responseMeta, responseErr, err
		}
		var response []any
		var attempt *recoveryAttempt
		for elem := range responseCh {
			response = append(response, elem)
			if responseError, ok := elem.(coreutils.SysError); ok {
				switch responseError.Message {
				case fmt.Sprintf("partition %d: %s", key.partitionID, errRecoveryInProgress):
					attempt, _ = s.control.partitions.latest(key)
				case fmt.Sprintf("workspace %d: %s", req.WSID, errRecoveryInProgress):
					attempt, _ = s.control.workspaces.latest(workspaceKey{partitionKey: key, wsid: req.WSID})
				}
			}
		}
		if attempt == nil {
			replay := make(chan any, len(response))
			for _, elem := range response {
				replay <- elem
			}
			close(replay)
			return replay, responseMeta, responseErr, nil
		}
		if *responseErr != nil {
			return nil, bus.ResponseMeta{}, nil, *responseErr
		}
		if err := attempt.wait(ctx); err != nil {
			return nil, bus.ResponseMeta{}, nil, err
		}
	}
}

func TestRecoveryRetrySender(t *testing.T) {
	key := partitionKey{appQName: testAppName, partitionID: 2}
	req := bus.Request{AppQName: testAppName, WSID: 7}
	workspaceErr := errors.New("workspace scan failed")
	for _, test := range []struct {
		name       string
		message    string
		wantErr    error
		wantStatus int
		wantCalls  int
	}{
		{name: "retries after partition recovery", message: "partition 2: recovery is in progress", wantStatus: http.StatusOK, wantCalls: 2},
		{name: "waits for matching workspace recovery", message: "workspace 7: recovery is in progress", wantErr: workspaceErr, wantCalls: 1},
		{name: "preserves unrelated 503", message: "no available engines", wantStatus: http.StatusServiceUnavailable, wantCalls: 1},
		{name: "preserves recovery limit response", message: "workspace 7: recovery concurrency limit is reached", wantStatus: http.StatusServiceUnavailable, wantCalls: 1},
		{name: "ignores another workspace", message: "workspace 8: recovery is in progress", wantStatus: http.StatusServiceUnavailable, wantCalls: 1},
		{name: "ignores another partition", message: "partition 3: recovery is in progress", wantStatus: http.StatusServiceUnavailable, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			control := newRecoveryTestControl()
			control.partitions.scheduled(key)
			control.partitions.attemptCompleted(key, nil)
			wsKey := workspaceKey{partitionKey: key, wsid: req.WSID}
			control.workspaces.scheduled(wsKey)
			control.workspaces.attemptCompleted(wsKey, workspaceErr)
			otherWSKey := workspaceKey{partitionKey: partitionKey{appQName: testAppName, partitionID: 3}, wsid: req.WSID}
			control.workspaces.scheduled(otherWSKey)
			control.workspaces.attemptCompleted(otherWSKey, errors.New("another partition's workspace scan failed"))

			responseError := coreutils.SysError{HTTPStatus: http.StatusServiceUnavailable, Message: test.message}
			calls := 0
			sender := &recoveryRetrySender{
				raw: bus.NewIRequestSender(testingu.MockTime, func(_ context.Context, _ bus.Request, responder bus.IResponder) {
					calls++
					if calls == 1 {
						bus.ReplyErr(responder, responseError)
					} else {
						bus.ReplyJSON(responder, http.StatusOK, "ok")
					}
				}),
				control: control,
				keyForRequest: func(bus.Request) (partitionKey, bool) {
					return key, true
				},
			}

			responseCh, meta, responseErr, err := sender.SendRequest(t.Context(), req)

			require.Equal(t, test.wantCalls, calls)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantStatus, meta.StatusCode)
			var response []any
			for elem := range responseCh {
				response = append(response, elem)
			}
			require.NoError(t, *responseErr)
			if test.wantStatus == http.StatusServiceUnavailable {
				require.Equal(t, []any{responseError}, response)
			} else {
				require.Equal(t, []any{"ok"}, response)
			}
		})
	}
}
