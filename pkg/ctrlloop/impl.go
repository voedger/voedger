/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/untillpro/goutils/logger"
)

func scheduler[Key comparable, SP any, State any](in chan OriginalMessage[Key, SP], dedupInCh chan statefulMessage[Key, SP, State], repeatCh chan scheduledMessage[Key, SP, State], nowTimeFunc nowTimeFunction) {
	defer close(dedupInCh)

	ScheduledItems := list.New()
	var serialNumber uint64

	timer := time.NewTimer(0)
	<-timer.C

	ok := true
	var m OriginalMessage[Key, SP]
	for ok {
		now := nowTimeFunc()
		select {
		case m, ok = <-in:
			if !ok {
				return
			}

			logger.Verbose(fmt.Sprintf("<-in. %v", m))

			serialNumber++
			item := scheduledMessage[Key, SP, State]{
				Key:          m.Key,
				SP:           m.SP,
				serialNumber: serialNumber,
				StartTime:    nextStartTimeFunc(m.CronSchedule, m.StartTimeTolerance, now),
			}

			addItemToSchedule(ScheduledItems, item)
			resetTimerToTop[Key, SP, State](timer, ScheduledItems, now)
		case <-timer.C:
			logger.Verbose(`<-timer.C`)

			element := ScheduledItems.Front()
			if element == nil {
				break
			}

			item := element.Value.(scheduledMessage[Key, SP, State])

			// non-blocking send to dedupIn
			select {
			case dedupInCh <- statefulMessage[Key, SP, State]{
				Key:          item.Key,
				SP:           item.SP,
				serialNumber: item.serialNumber,
			}:
				ScheduledItems.Remove(element)
			default:
			}

			resetTimerToTop[Key, SP, State](timer, ScheduledItems, now)
		case m := <-repeatCh:
			logger.Verbose(fmt.Sprintf("<-repeatCh. %v", m))

			addItemToSchedule(ScheduledItems, m)
			resetTimerToTop[Key, SP, State](timer, ScheduledItems, now)
		}
	}
}

func resetTimerToTop[Key comparable, SP any, State any](timer *time.Timer, l *list.List, now time.Time) {
	item := l.Front()
	if item == nil {
		return
	}

	resetTimer(timer, item.Value.(scheduledMessage[Key, SP, State]).StartTime.Sub(now))
}

// addItemToSchedule adds new item to schedule
func addItemToSchedule[Key comparable, SP any, State any](l *list.List, m scheduledMessage[Key, SP, State]) {
	logger.Log(1, logger.LogLevelVerbose, m.String())
	// serial number controlling
	for element := l.Front(); element != nil; element = element.Next() {
		currentScheduledMessage := element.Value.(scheduledMessage[Key, SP, State])
		if m.Key == currentScheduledMessage.Key {
			if m.serialNumber > currentScheduledMessage.serialNumber {
				l.Remove(element)
				break
			}
			// skip scheduling old serial number
			logger.Log(1, logger.LogLevelVerbose, fmt.Sprintf("skipped old serialNumber of the message: %v", m))
			return
		}
	}
	// scheduling is going here
	index := 0
	for element := l.Front(); element != nil; element = element.Next() {
		if m.StartTime.Before(element.Value.(scheduledMessage[Key, SP, State]).StartTime) {
			l.InsertBefore(m, element)
			return
		}
		index++
	}

	l.PushBack(m)
}

func dedupIn[Key comparable, SP any, State any](in chan statefulMessage[Key, SP, State], callerCh chan statefulMessage[Key, SP, State], repeatCh chan scheduledMessage[Key, SP, State], InProcess *sync.Map, nowTimeFunc nowTimeFunction) {
	defer close(callerCh)

	for m := range in {
		logger.Verbose(m.String())

		if _, ok := InProcess.Load(m.Key); ok {
			repeatCh <- scheduledMessage[Key, SP, State]{
				Key:          m.Key,
				SP:           m.SP,
				serialNumber: m.serialNumber,
				StartTime:    nowTimeFunc().Add(DedupScheduleInterval),
			}
			continue
		}
		InProcess.Store(m.Key, struct{}{})
		callerCh <- m
	}
}

func dedupOut[Key comparable, SP any, PV any, State any](in chan answer[Key, SP, PV, State], repeaterCh chan answer[Key, SP, PV, State], InProcess *sync.Map) {
	defer close(repeaterCh)

	for m := range in {
		logger.Verbose(m.String())

		InProcess.Delete(m.Key)
		repeaterCh <- m
	}
}

func caller[Key comparable, SP any, PV any, State any](in chan statefulMessage[Key, SP, State], dedupOutCh chan answer[Key, SP, PV, State], callerFinalizerCh chan struct{}, controllerFunc ControllerFunction[Key, SP, State, PV]) {
	defer func() {
		select {
		case <-callerFinalizerCh:
			break
		default:
			close(dedupOutCh)
			close(callerFinalizerCh)
		}
	}()

	for m := range in {
		logger.Verbose(m.String())

		newState, pv, startTime := controllerFunc(m.Key, m.SP, m.State)
		dedupOutCh <- answer[Key, SP, PV, State]{
			Key:          m.Key,
			SP:           m.SP,
			serialNumber: m.serialNumber,
			State:        newState,
			PV:           pv,
			StartTime:    startTime,
		}
	}
}

func repeater[Key comparable, SP any, PV any, State any](in chan answer[Key, SP, PV, State], repeatCh chan scheduledMessage[Key, SP, State], reporterCh chan reportInfo[Key, PV]) {
	defer close(repeatCh)
	defer close(reporterCh)

	for m := range in {
		logger.Verbose(m.String())

		if m.StartTime != nil {
			repeatCh <- scheduledMessage[Key, SP, State]{
				Key:          m.Key,
				SP:           m.SP,
				serialNumber: m.serialNumber,
				StartTime:    *m.StartTime,
			}
		}
		if m.PV != nil {
			reporterCh <- reportInfo[Key, PV]{
				Key: m.Key,
				PV:  m.PV,
			}
		}
	}
}

func reporter[Key comparable, PV any](in chan reportInfo[Key, PV], reporterFunc ReporterFunction[Key, PV]) {
	ToBeReported := list.New()
	timer := time.NewTimer(0)
	<-timer.C

	ok := true
	var m reportInfo[Key, PV]
	for ok {
		select {
		case m, ok = <-in:
			if !ok {
				return
			}
			logger.Verbose(m.String())

			if err := reporterFunc(m.Key, m.PV); err != nil {
				ToBeReported.PushBack(reportInfoAttempt[Key, PV]{
					Key:     m.Key,
					PV:      m.PV,
					Attempt: 1,
				})
				resetTimer(timer, ReportInterval)
			}
		case <-timer.C:
			element := ToBeReported.Front()
			if element == nil {
				break
			}

			r := element.Value.(reportInfoAttempt[Key, PV])
			ToBeReported.Remove(element)

			if r.Attempt < MaxReportAttemptNumber {
				if err := reporterFunc(r.Key, r.PV); err != nil {
					ToBeReported.PushBack(reportInfoAttempt[Key, PV]{
						Key:     r.Key,
						PV:      r.PV,
						Attempt: r.Attempt + 1,
					})
				}
			}
			resetTimer(timer, ReportInterval)
		}
	}
}
