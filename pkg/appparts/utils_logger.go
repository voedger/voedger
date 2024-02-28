/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"
	"sync"
	"time"

	"github.com/untillpro/goutils/logger"
)

type intervalLogger struct {
	interval time.Duration
	mx       sync.Mutex
	errs     map[string]time.Time
}

func (l *intervalLogger) error(err error) {
	e := fmt.Sprint(err)

	l.mx.Lock()
	if t, ok := l.errs[e]; ok {
		if time.Since(t) < l.interval {
			return
		}
	}
	l.errs[e] = time.Now()
	l.mx.Unlock()

	logger.Error(e)
}

var minuteLogger = &intervalLogger{
	interval: time.Minute,
	mx:       sync.Mutex{},
	errs:     map[string]time.Time{},
}
