/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"container/list"
	"fmt"
	"time"

	"github.com/aptible/supercronic/cronexpr"
	"github.com/untillpro/goutils/logger"
)

func getNextStartTime(cronSchedule string, startTimeTolerance time.Duration, now time.Time) time.Time {
	nextStartExp, err := cronexpr.Parse(cronSchedule)
	if err != nil {
		logger.Verbose(fmt.Sprintf("wrong CronSchedule field = %s", cronSchedule))
		return now
	}
	return nextStartExp.Next(now.Add(-startTimeTolerance))
}

func resetTimer(timer *time.Timer, d time.Duration) {
	logger.Verbose(fmt.Sprintf("resetTimer. Start after: %v", d))
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
			timer.Stop()
		}
	}
	logger.Verbose("timer restarted!")
	timer.Reset(d)
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
