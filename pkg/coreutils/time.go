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
	NewTimerChan(d time.Duration) <-chan time.Time
}

// MockTime must be a global var to avoid case when different times could be used in tests.
// jwt.TimeFunc is a global var: once set it must not be changed during tests
var MockTime IMockTime = &mockedTime{
	now:     time.Now().Add(-5 * time.Minute), // decrease current time to avoid "token used before issued" error in bp3 utils_test.go
	RWMutex: sync.RWMutex{},
	timers:  map[mockTimer]struct{}{},
}

type IMockTime interface {
	ITime

	// implementation must trigger each timer created by IMockTime.NewTimer() if the time has come after adding
	Add(d time.Duration)
}

func NewITime() ITime {
	return &realTime{}
}

type realTime struct{}

type mockedTime struct {
	sync.RWMutex
	now    time.Time
	timers map[mockTimer]struct{}
}

func (t *realTime) Now() time.Time {
	return time.Now()
}

func (t *realTime) NewTimerChan(d time.Duration) <-chan time.Time {
	res := time.NewTimer(d)
	return res.C
}

func (t *mockedTime) Now() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.now
}

func (t *mockedTime) NewTimerChan(d time.Duration) <-chan time.Time {
	t.Lock()
	defer t.Unlock()
	mt := mockTimer{
		c:          make(chan time.Time, 1),
		expiration: t.now.Add(d),
	}
	t.timers[mt] = struct{}{}
	return mt.c
}

type mockTimer struct {
	c          chan time.Time
	expiration time.Time
}

func (t *mockedTime) Add(d time.Duration) {
	t.Lock()
	defer t.Unlock()
	t.now = t.now.Add(d)
	t.checkTimers()
}

func (t *mockedTime) checkTimers() {
	for timer := range t.timers {
		if t.now.Equal(timer.expiration) || t.now.After(timer.expiration) {
			timer.c <- t.now
			delete(t.timers, timer)
		}
	}
}
