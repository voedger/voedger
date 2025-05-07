/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package timeu

import (
	"time"
)

type ITime interface {
	Now() time.Time
	NewTimerChan(d time.Duration) <-chan time.Time
	Sleep(d time.Duration)
}

func NewITime() ITime {
	return &realTime{}
}

type realTime struct{}

func (t *realTime) Now() time.Time {
	return time.Now()
}

func (t *realTime) NewTimerChan(d time.Duration) <-chan time.Time {
	res := time.NewTimer(d)
	return res.C
}

func (t *realTime) Sleep(d time.Duration) {
	time.Sleep(d)
}
