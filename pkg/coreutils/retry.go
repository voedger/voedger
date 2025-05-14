/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"context"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
)

// Retry attempts to execute f() until it accomplished without error
// f() returns error -> error is logged, try again after 500ms
// ctx is cancelled during retires -> context.Canceled is returned
func Retry(ctx context.Context, iTime timeu.ITime, f func() error) error {
	var lastErr error
	for ctx.Err() == nil {
		if lastErr = f(); lastErr == nil {
			return nil
		}
		logger.Error(lastErr)
		timerCh := iTime.NewTimerChan(defaultRetryDelay)
		select {
		case <-ctx.Done():
			break
		case <-timerCh:
		}
	}
	return ctx.Err()
}
