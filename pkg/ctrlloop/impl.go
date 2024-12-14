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

	"github.com/voedger/voedger/pkg/goutils/logger"
)

func scheduler[Key comparable, SP any, State any](in chan ControlMessage[Key, SP], dedupInCh chan statefulMessage[Key, SP, State], repeatCh chan scheduledMessage[Key, SP, State], nowTimeFunc nowTimeFunction) {
	defer close(dedupInCh)

	schedulerObj := newScheduler[Key, SP, State](nil)
	var serialNumber uint64

	ok := true
	var m ControlMessage[Key, SP]
	for ok {
		now := nowTimeFunc()
		select {
		case m, ok = <-in:
			if !ok {
				return
			}
			logger.Verbose(fmt.Sprintf("<-in. %v", m))

			serialNumber++
			schedulerObj.OnIn(serialNumber, m, now)
		case <-schedulerObj.Tick():
			logger.Verbose(`<-timer.C`)

			schedulerObj.OnTimer(dedupInCh, now)
		case m := <-repeatCh:
			logger.Verbose(fmt.Sprintf("<-repeatCh. %v", m))

			schedulerObj.OnRepeat(m, now)
		}
	}
}

func dedupIn[Key comparable, SP any, State any](in chan statefulMessage[Key, SP, State], callerCh chan statefulMessage[Key, SP, State], repeatCh chan scheduledMessage[Key, SP, State], inProcess *sync.Map, nowTimeFunc nowTimeFunction) {
	defer close(callerCh)

	for m := range in {
		logger.Verbose(m.String())

		if _, ok := inProcess.Load(m.Key); ok {
			repeatCh <- scheduledMessage[Key, SP, State]{
				Key:          m.Key,
				SP:           m.SP,
				serialNumber: m.serialNumber,
				StartTime:    nowTimeFunc().Add(DedupScheduleInterval),
			}
			continue
		}
		inProcess.Store(m.Key, struct{}{})
		callerCh <- m
	}
}

func dedupOut[Key comparable, SP any, PV any, State any](in chan answer[Key, SP, PV, State], repeaterCh chan answer[Key, SP, PV, State], inProcess *sync.Map) {
	defer close(repeaterCh)

	for m := range in {
		logger.Verbose(m.String())

		inProcess.Delete(m.Key)
		repeaterCh <- m
	}
}

func caller[Key comparable, SP any, PV any, State any](in chan statefulMessage[Key, SP, State], dedupOutCh chan answer[Key, SP, PV, State], callerFinalizerCh chan struct{}, controllerFunc ControllerFunction[Key, SP, State, PV]) {
	defer func() {
		if _, ok := <-callerFinalizerCh; !ok {
			close(dedupOutCh)
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

func reporter[Key comparable, PV any](in chan reportInfo[Key, PV], finishCh chan<- struct{}, reporterFunc ReporterFunction[Key, PV]) {
	defer func() {
		finishCh <- struct{}{}
	}()

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
