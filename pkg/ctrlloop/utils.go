/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"fmt"
	"time"

	"github.com/aptible/supercronic/cronexpr"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func getNextStartTime(cronSchedule string, startTimeTolerance time.Duration, now time.Time) time.Time {
	nextStartExp, err := cronexpr.Parse(cronSchedule)
	if err != nil {
		logger.Verbose("wrong CronSchedule field = " + cronSchedule)
		return now
	}
	return nextStartExp.Next(now.Add(-startTimeTolerance))
}

func resetTimer(timer *time.Timer, d time.Duration) {
	logger.Verbose(fmt.Sprintf("resetTimer. Start after: %v", d))
	timer.Reset(d)
	logger.Verbose("timer restarted!")
}
