/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package testingu

import (
	"fmt"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/goutils/timeu"
)

// MockTime must be a global var to avoid case when different times could be used in tests.
var MockTime = NewMockTime()

type IMockTime interface {
	timeu.ITime

	// implementation must trigger each timer created by IMockTime.NewTimer() if the time has come after adding
	Add(d time.Duration)

	// next timer got by NewTimerChan already will contain firing
	// useful when we do not know the instant when NewTimer() will be called but we advancing the time to make it fire
	FireNextTimerImmediately()

	SetOnNextNewTimerChan(f func())
	SetOnNextTimerArmed(f func())

	// NewIsolatedTime creates a new isolated MockTime instance that starts from the same time
	// but advances independently from the original MockTime.
	// Used to isolate scheduler time from global MockTime in tests.
	NewIsolatedTime() timeu.ITime
}

func NewMockTime() IMockTime {
	return &mockedTime{
		now:     time.Now(),
		RWMutex: sync.RWMutex{},
		timers:  map[mockTimer]struct{}{},
	}
}

type mockedTime struct {
	sync.RWMutex
	now                      time.Time
	timers                   map[mockTimer]struct{}
	fireNextTimerImmediately bool
	onNextNewTimerChan       func()
	onNextTimerArmed         func()
}

func (t *mockedTime) Now() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.now
}

func (t *mockedTime) NewTimerChan(d time.Duration) <-chan time.Time {
	t.Lock()
	defer t.Unlock()
	if t.onNextNewTimerChan != nil {
		t.onNextNewTimerChan()
	}
	mt := mockTimer{
		c:          make(chan time.Time, 1),
		expiration: t.now.Add(d),
	}
	if t.onNextTimerArmed != nil {
		t.onNextTimerArmed()
	}
	t.timers[mt] = struct{}{}
	if t.fireNextTimerImmediately {
		mt.c <- t.now
		t.fireNextTimerImmediately = false
	}
	return mt.c
}

func (t *mockedTime) FireNextTimerImmediately() {
	t.Lock()
	t.fireNextTimerImmediately = true
	t.Unlock()
}

func (t *mockedTime) SetOnNextNewTimerChan(f func()) {
	t.Lock()
	t.onNextNewTimerChan = func() {
		f()
		t.onNextNewTimerChan = nil
	}
	t.Unlock()
}

func (t *mockedTime) SetOnNextTimerArmed(f func()) {
	t.Lock()
	t.onNextTimerArmed = func() {
		f()
		t.onNextTimerArmed = nil
	}
	t.Unlock()
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

func (t *mockedTime) Sleep(d time.Duration) {
	t.Add(d)
}

func (t *mockedTime) checkTimers() {
	for timer := range t.timers {
		if t.now.Equal(timer.expiration) || t.now.After(timer.expiration) {
			timer.c <- t.now
			delete(t.timers, timer)
		}
	}
}

// String implements fmt.Stringer to prevent reflection-based unprotected access to internal fields
// if mockedTime is used as a mock argument then the mock engine will compare fmt.Sprintf("%v", expectedArg) and fmt.Sprintf("%v", actualArg) to check expectations
// where actualArg is *mockedTime
// that could lead to data race: fmt.Sprintf() reads fields of mockedTime via reflection without protection
// whereas someone calls mockedTime methods that writes internal fields (protected via mutex)
// String method exists -> fmt.Sprintf() will use it instead of reflection
func (t *mockedTime) String() string {
	t.RLock()
	defer t.RUnlock()
	return fmt.Sprintf("mockedTime{now=%s, timers=%d, fireNextTimerImmediately=%t}",
		t.now.Format(time.RFC3339Nano), len(t.timers), t.fireNextTimerImmediately)
}

// NewIsolatedTime creates a new isolated MockTime instance that starts from the same time
// but advances independently from the original MockTime.
func (t *mockedTime) NewIsolatedTime() timeu.ITime {
	return NewMockTime()
}
