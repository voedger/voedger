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
)

// partitionRecoveryHooks provides deterministic synchronization points for package tests.
// Production command processors uses nopHooks()
type partitionRecoveryHooks struct {
	scheduled        func(partitionKey)
	beforeAttempt    func(context.Context, partitionKey) error
	attemptCompleted func(partitionKey, error)
}

type recoveryAttempt struct {
	done chan struct{}
	gate <-chan struct{}
	err  error
}

type recoveryTestControl struct {
	mu           sync.Mutex
	attempts     map[partitionKey]*recoveryAttempt
	starts       map[partitionKey]int
	nextGates    map[partitionKey]<-chan struct{}
	nextFailures map[partitionKey]error
}

func newRecoveryTestControl() *recoveryTestControl {
	return &recoveryTestControl{
		attempts:     map[partitionKey]*recoveryAttempt{},
		starts:       map[partitionKey]int{},
		nextGates:    map[partitionKey]<-chan struct{}{},
		nextFailures: map[partitionKey]error{},
	}
}

func (c *recoveryTestControl) testHooks() *partitionRecoveryHooks {
	return &partitionRecoveryHooks{
		scheduled:        c.recoveryStarted,
		beforeAttempt:    c.beforeRecovery,
		attemptCompleted: c.recoveryFinished,
	}
}

func (c *recoveryTestControl) recoveryStarted(key partitionKey) {
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

func (c *recoveryTestControl) beforeRecovery(ctx context.Context, key partitionKey) error {
	c.mu.Lock()
	attempt := c.attempts[key]
	c.mu.Unlock()
	if attempt.gate != nil {
		select {
		case <-attempt.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return attempt.err
}

func (c *recoveryTestControl) recoveryFinished(key partitionKey, err error) {
	c.mu.Lock()
	attempt := c.attempts[key]
	attempt.err = err
	close(attempt.done)
	c.mu.Unlock()
}

func (c *recoveryTestControl) blockNext(key partitionKey) chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	gate := make(chan struct{})
	c.nextGates[key] = gate
	return gate
}

func (c *recoveryTestControl) failNext(key partitionKey, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextFailures[key] = err
}

func (c *recoveryTestControl) latestAttempt(key partitionKey) (*recoveryAttempt, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	attempt, ok := c.attempts[key]
	return attempt, ok
}

func (c *recoveryTestControl) wait(ctx context.Context, key partitionKey) error {
	attempt, ok := c.latestAttempt(key)
	if !ok {
		return fmt.Errorf("recovery for %s partition %d was not started", key.appQName, key.partitionID)
	}
	select {
	case <-attempt.done:
		return attempt.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *recoveryTestControl) startCount(key partitionKey) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.starts[key]
}

type recoveryRetrySender struct {
	raw           bus.IRequestSender
	control       *recoveryTestControl
	keyForRequest func(bus.Request) (partitionKey, bool)
}

func (s *recoveryRetrySender) SendRequest(ctx context.Context, req bus.Request) (<-chan any, bus.ResponseMeta, *error, error) {
	for {
		responseCh, responseMeta, responseErr, err := s.raw.SendRequest(ctx, req)
		key, retryRecovery := s.keyForRequest(req)
		if err != nil || responseMeta.StatusCode != http.StatusServiceUnavailable || !retryRecovery {
			return responseCh, responseMeta, responseErr, err
		}
		if _, ok := s.control.latestAttempt(key); !ok {
			return responseCh, responseMeta, responseErr, err
		}
		for range responseCh {
		}
		if *responseErr != nil {
			return nil, bus.ResponseMeta{}, nil, *responseErr
		}
		if err := s.control.wait(ctx, key); err != nil {
			return nil, bus.ResponseMeta{}, nil, err
		}
	}
}
