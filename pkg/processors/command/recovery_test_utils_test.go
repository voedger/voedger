/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/istructs"
)

// logReadFromTo turns one production log-reader call into a comparable test value:
//
//	ReadPLog/ReadWLog(offset, count) --> logReadFromTo{offset, count}
//
// Tests use the resulting sequence to prove that recovery starts at its
// checkpoint and performs one inclusive scan to the end.
type logReadFromTo struct {
	offset istructs.Offset
	count  int
}

// logReadsRecorder solves concurrent observation of recovery log reads:
//
//	recovery goroutine --record(key, read)--\
//	                                          +--> mutex --> byKey[key] history
//	test goroutine --------values(key)--------/                  |
//	                                                             +--> copied snapshot
//
// The lock keeps test inspection race-free without adding synchronization to
// production PLog or WLog readers.
type logReadsRecorder[K comparable] struct {
	mu    sync.Mutex
	byKey map[K][]logReadFromTo
}

func newLogReadsRecorder[K comparable]() *logReadsRecorder[K] {
	return &logReadsRecorder[K]{byKey: map[K][]logReadFromTo{}}
}

// record is the production hook at the write side of the observation flow:
// Append order preserves the exact log-read order for that key.
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

// values returns recorded log records snapshot
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

// attemptCompleted publishes the manager's final result to waiting tests:
//
//	final wrapped error --> attempt.err --> close(done) --> wait(key)
func (c *recoveryTestAttempts[K, T]) attemptCompleted(key K, err error) {
	c.mu.Lock()
	attempt := c.attempts[key]
	attempt.err = err
	close(attempt.done)
	c.mu.Unlock()
}

// blockNext prepares a one-shot scheduling barrier:
//
//	blockNext(key) --> gate --> next scheduled attempt --> waits for close(gate)
//
// Holding the gate open makes in-progress and concurrency-limit assertions
// independent of goroutine timing.
func (c *recoveryTestAttempts[K, T]) blockNext(key K) chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	gate := make(chan struct{})
	c.nextGates[key] = gate
	return gate
}

// failNext prepares a one-shot failure before production recovery runs:
//
//	failNext(key, err) --> next scheduled attempt --> err --> retained failure
//
// This isolates retry tests from storage or log-reader failures.
func (c *recoveryTestAttempts[K, T]) failNext(key K, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextFailures[key] = err
}

// latest distinguishes a recoverable test 503 from an unrelated 503:
//
//	attempts[key] present --> caller may wait and retry
//	attempts[key] absent  --> caller preserves the response
func (c *recoveryTestAttempts[K, T]) latest(key K) (*recoveryAttempt, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	attempt, ok := c.attempts[key]
	return attempt, ok
}

// wait joins the latest known attempt without sleeps or polling:
//
//	attempt.done closed --> return attempt.err
//	ctx canceled ---------> return ctx.Err()
//	no attempt -----------> return "not started"
func (c *recoveryTestAttempts[K, T]) wait(ctx context.Context, key K) error {
	attempt, ok := c.latest(key)
	if !ok {
		return fmt.Errorf("recovery for %v was not started", key)
	}
	select {
	case <-attempt.done:
		return attempt.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// startCount exposes admission as a stable assertion:
//
//	requests for key --> recoveryManager --> scheduled calls --> starts[key]
//
// Deduplicated and concurrency-rejected requests do not increase the count.
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

// newRecoveryTestControl creates four empty, independently locked channels:
//
//	{partition attempts, workspace attempts, PLog reads, WLog reads}
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

// recoveryRetrySender hides mandatory lazy recovery from command tests whose
// subject is unrelated to recovery. For requests selected by keyForRequest, the
// fixture treats a 503 as recoverable only when that partition has a recorded
// attempt; rawRequestSender remains available to tests that need to assert 503s
// directly instead of applying this test-only assumption.
type recoveryRetrySender struct {
	raw           bus.IRequestSender
	control       *recoveryTestControl
	keyForRequest func(bus.Request) (partitionKey, bool)
}

// SendRequest hides mandatory lazy partition recovery from unrelated tests:
//
//	request --> raw.SendRequest
//	              |
//	              +--> non-503/error/untracked request ----------> return response
//	              |
//	              +--> 503 + known partition attempt
//	                         |
//	                         +--> drain response --> wait(done) --> retry request
//	                                                  |
//	                                                  +--> failure/cancel --> return error
//
// Draining before waiting preserves the bus response lifecycle. Checking for a
// recorded attempt prevents unrelated 503 responses from becoming retry loops.
func (s *recoveryRetrySender) SendRequest(ctx context.Context, req bus.Request) (<-chan any, bus.ResponseMeta, *error, error) {
	for {
		responseCh, responseMeta, responseErr, err := s.raw.SendRequest(ctx, req)
		key, retryRecovery := s.keyForRequest(req)
		if err != nil || responseMeta.StatusCode != http.StatusServiceUnavailable || !retryRecovery {
			return responseCh, responseMeta, responseErr, err
		}
		if _, ok := s.control.partitions.latest(key); !ok {
			return responseCh, responseMeta, responseErr, err
		}
		for range responseCh {
		}
		if *responseErr != nil {
			return nil, bus.ResponseMeta{}, nil, *responseErr
		}
		if err := s.control.partitions.wait(ctx, key); err != nil {
			return nil, bus.ResponseMeta{}, nil, err
		}
	}
}
