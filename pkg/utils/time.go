/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"sync"
	"time"
)

type ITime interface {
	Now() time.Time
	NewTimer(d time.Duration) <-chan time.Time
}

var MockTime IMockTime = &mockedTime{
	now:     time.Now(),
	RWMutex: sync.RWMutex{},
}

type IMockTime interface {
	ITime
	Add(d time.Duration)
}

type realTime struct{}

type mockedTime struct {
	sync.RWMutex
	now    time.Time
	timers sync.Map
}

func NewITime() ITime {
	return &realTime{}
}

func (t *realTime) Now() time.Time {
	return time.Now()
}

func (t *realTime) NewTimer(d time.Duration) <-chan time.Time {
	res := time.NewTimer(d)
	return res.C
}

func (t *mockedTime) Now() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.now
}

func (t *mockedTime) NewTimer(d time.Duration) <-chan time.Time {
	mt := &MockTimer{
		C:          make(chan time.Time, 1),
		expiration: t.now.Add(d),
	}
	// Store the timer in the registry
	t.timers.Store(mt, struct{}{})
	return mt.C
}

type MockTimer struct {
	C          chan time.Time
	expiration time.Time
	fired      bool
}

func (t *mockedTime) Add(d time.Duration) {
	t.Lock()
	defer t.Unlock()
	t.now = t.now.Add(d)
	t.checkTimers()
}

func (t *mockedTime) checkTimers() {
	t.timers.Range(func(key, value any) bool {
		timer := key.(*MockTimer)
		if !timer.fired && (t.now.Equal(timer.expiration) || t.now.After(timer.expiration)) {
			timer.fired = true
			select {
			case timer.C <- t.now:
			default:
			}
			t.timers.Delete(timer)
		}
		return true
	})
}
