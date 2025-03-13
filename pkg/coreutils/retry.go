/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"context"
	"time"
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
func Retry(ctx context.Context, iTime ITime, retryDelay time.Duration, retryCount int, f func() error) error {
	var err error

	for i := 1; retryCount == 0 || i <= retryCount; i++ {
		timerCh := iTime.NewTimerChan(retryDelay)
		if err = f(); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return err
		case <-timerCh:
		}
	}

	return ErrRetryAttemptsExceeded
}
