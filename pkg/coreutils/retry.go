/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"context"
	"time"
)

func Retry(ctx context.Context, iTime ITime, retryDelay time.Duration, retryCount int, f func() error) error {
	var err error

	for i := 0; i < retryCount; i++ {
		if err = f(); err == nil {
			return nil
		}

		timerCh := iTime.NewTimerChan(retryDelay)
		select {
		case <-ctx.Done():
			return err
		case <-timerCh:
		}
	}

	return ErrRetryAttemptsExceeded
}
