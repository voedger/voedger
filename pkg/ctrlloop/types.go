/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"container/list"
	"time"
)

type (
	ControllerFunction[Key comparable, SP any, State any, PV any] func(key Key, sp SP, state State) (newState *State, pv *PV, startTime *time.Time)
	ReporterFunction[Key comparable, PV any]                      func(key Key, pv *PV) (err error)
	nowTimeFunction                                               func() time.Time
)

type schedulerImp[Key comparable, SP any, State any] struct {
	timer          *time.Timer
	ScheduledItems *list.List
}

func newScheduler[Key comparable, SP any, State any](l *list.List) *schedulerImp[Key, SP, State] {
	timer := time.NewTimer(0)
	<-timer.C

	if l == nil {
		l = list.New()
	}
	return &schedulerImp[Key, SP, State]{
		timer:          timer,
		ScheduledItems: l,
	}
}

func (s *schedulerImp[Key, SP, State]) Tick() <-chan time.Time {
	return s.timer.C
}

func (s *schedulerImp[Key, SP, State]) OnIn(serialNumber uint64, m OriginalMessage[Key, SP], now time.Time) {
	item := scheduledMessage[Key, SP, State]{
		Key:          m.Key,
		SP:           m.SP,
		serialNumber: serialNumber,
		StartTime:    nextStartTimeFunc(m.CronSchedule, m.StartTimeTolerance, now),
	}

	addItemToSchedule(s.ScheduledItems, item)
	resetTimerToTop[Key, SP, State](s.timer, s.ScheduledItems, now)
}

func (s *schedulerImp[Key, SP, State]) OnTimer(dedupInCh chan<- statefulMessage[Key, SP, State], now time.Time) {
	element := s.ScheduledItems.Front()
	if element == nil {
		return
	}

	item := element.Value.(scheduledMessage[Key, SP, State])

	// non-blocking send to dedupIn
	select {
	case dedupInCh <- statefulMessage[Key, SP, State]{
		Key:          item.Key,
		SP:           item.SP,
		serialNumber: item.serialNumber,
	}:
		s.ScheduledItems.Remove(element)
	default:
	}

	resetTimerToTop[Key, SP, State](s.timer, s.ScheduledItems, now)
}

func (s *schedulerImp[Key, SP, State]) OnRepeat(m scheduledMessage[Key, SP, State], now time.Time) {
	addItemToSchedule(s.ScheduledItems, m)
	resetTimerToTop[Key, SP, State](s.timer, s.ScheduledItems, now)
}
