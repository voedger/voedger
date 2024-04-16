/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"container/list"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

type (
	ControllerFunction[Key comparable, SP any, State any, PV any] func(key Key, sp SP, state State) (newState *State, pv *PV, startTime *time.Time)
	ReporterFunction[Key comparable, PV any]                      func(key Key, pv *PV) (err error)
	nowTimeFunction                                               func() time.Time
	nextStartTimeFunction                                         func(cronSchedule string, startTimeTolerance time.Duration, now time.Time) time.Time
)

type schedulerImp[Key comparable, SP any, State any] struct {
	timer          *time.Timer
	scheduledItems *list.List
	lastDuration   time.Duration
}

func newScheduler[Key comparable, SP any, State any](l *list.List) *schedulerImp[Key, SP, State] {
	timer := time.NewTimer(0)
	<-timer.C

	if l == nil {
		l = list.New()
	}
	return &schedulerImp[Key, SP, State]{
		timer:          timer,
		scheduledItems: l,
	}
}

func (s *schedulerImp[Key, SP, State]) Tick() <-chan time.Time {
	return s.timer.C
}

func (s *schedulerImp[Key, SP, State]) OnIn(serialNumber uint64, m ControlMessage[Key, SP], now time.Time) {
	item := scheduledMessage[Key, SP, State]{
		Key:          m.Key,
		SP:           m.SP,
		serialNumber: serialNumber,
		StartTime:    nextStartTimeFunc(m.CronSchedule, m.StartTimeTolerance, now),
	}

	s.AddItemToSchedule(item)
	s.ResetTimerToTop(s.scheduledItems, now)
}

func (s *schedulerImp[Key, SP, State]) OnTimer(dedupInCh chan<- statefulMessage[Key, SP, State], now time.Time) {
	element := s.scheduledItems.Front()
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
		s.scheduledItems.Remove(element)
		s.ResetTimerToTop(s.scheduledItems, now)
	default:
		s.ResetTimer(DedupInRetryInterval)
	}
}

func (s *schedulerImp[Key, SP, State]) OnRepeat(m scheduledMessage[Key, SP, State], now time.Time) {
	s.AddItemToSchedule(m)
	s.ResetTimerToTop(s.scheduledItems, now)
}

func (s *schedulerImp[Key, SP, State]) ResetTimer(d time.Duration) {
	s.lastDuration = d
	resetTimer(s.timer, d)
}

func (s *schedulerImp[Key, SP, State]) ResetTimerToTop(l *list.List, now time.Time) {
	item := l.Front()
	if item == nil {
		return
	}
	s.ResetTimer(item.Value.(scheduledMessage[Key, SP, State]).StartTime.Sub(now))
}

func (s *schedulerImp[Key, SP, State]) AddItemToSchedule(m scheduledMessage[Key, SP, State]) {
	logger.Log(1, logger.LogLevelVerbose, m.String())
	// serial number controlling
	for element := s.scheduledItems.Front(); element != nil; element = element.Next() {
		currentScheduledMessage := element.Value.(scheduledMessage[Key, SP, State])
		if m.Key == currentScheduledMessage.Key {
			if m.serialNumber > currentScheduledMessage.serialNumber {
				s.scheduledItems.Remove(element)
				break
			}
			// skip scheduling old serial number
			logger.Log(1, logger.LogLevelVerbose, fmt.Sprintf("skipped old serialNumber of the message: %v", m))
			return
		}
	}
	// scheduling is going here
	for element := s.scheduledItems.Front(); element != nil; element = element.Next() {
		if m.StartTime.Before(element.Value.(scheduledMessage[Key, SP, State]).StartTime) {
			s.scheduledItems.InsertBefore(m, element)
			return
		}
	}

	s.scheduledItems.PushBack(m)
}
