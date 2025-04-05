/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

// FIXME: Cover with tests
// Retry attempts to execute the function f until it returns nil or the context is done.
// Parameters:
// ctx - context
// iTime - time interface
// retryDelay - delay between retries
// retryCount - number of retries (0 for infinite retries)
// f - function to execute
// Returns:
// error - error returned by the function f

// Retry attempts to execute f() until it acomplised without error
// f() returns error -> error is logged, try again after retryDelay
// retryDelay == 0 -> no delay
func Retry_(ctx context.Context, iTime ITime, retryDelay time.Duration, f func() error) error {
	var lastErr error
	for ctx.Err() == nil {
		if lastErr = f(); lastErr == nil {
			return nil
		}
		if retryDelay > 0 {
			logger.Error(lastErr)
			timerCh := iTime.NewTimerChan(retryDelay)
			select {
			case <-ctx.Done():
				return lastErr
			case <-timerCh:
			}
		} else {
			break
		}
	}
	return lastErr
}

func Retry(ctx context.Context, iTime ITime, retryDelay time.Duration, retryCount int, f func() error) error {
	var lastErr error

	for i := 1; retryCount == 0 || i <= retryCount; i++ {
		timerCh := iTime.NewTimerChan(retryDelay)
		if lastErr = f(); lastErr == nil {
			return nil
		}

		logger.Verbose(lastErr)

		select {
		case <-ctx.Done():
			return lastErr
		case <-timerCh:
		}
	}

	return lastErr
}
