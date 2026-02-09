/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

import (
	"context"
	"os"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
)

func newKillerScheduler(clock timeu.ITime) *killerScheduler {
	if p, ok := clock.(interface{ NewIsolatedTime() timeu.ITime }); ok {
		// happens in tests only
		// need to avoid killer firing on global MockTime advance
		clock = p.NewIsolatedTime()
	}
	return &killerScheduler{clock: clock}
}

func (ks *killerScheduler) scheduleKiller(deadline time.Time) {
	duration := deadline.Sub(ks.clock.Now())
	if duration <= 0 {
		return
	}
	if ks.ctx != nil {
		ks.cancel()
	}
	ks.ctx, ks.cancel = context.WithCancel(context.Background())
	go func(ctx context.Context) {
		timer := ks.clock.NewTimerChan(duration)
		select {
		case <-timer:
			logger.Error("killer timer fired -> terminating...")
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}(ks.ctx)
}
