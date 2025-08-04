/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @author Denis Gribanov
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier

import (
	"context"
	"errors"
	"time"
)

// Retry executes fn with retry logic and returns its result or an error.
func Retry[T any](ctx context.Context, cfg Config, op func() (T, error)) (T, error) {
	r, err := New(cfg)
	var zero T
	if err != nil {
		return zero, err
	}
	var result T
	err = r.Run(ctx, func() error {
		var fnErr error
		result, fnErr = op()
		return fnErr
	})
	return result, err
}

func RetryErr(ctx context.Context, cfg Config, op func() error) error {
	_, err := Retry(ctx, cfg, func() (any, error) {
		return nil, op()
	})
	return err
}

func RetryFor(ctx context.Context, cfg Config, maxElapsed time.Duration, op func() error) (ok bool, err error) {
	dlCtx, cancel := context.WithDeadline(ctx, time.Now().Add(maxElapsed))
	defer cancel()
	err = RetryErr(dlCtx, cfg, op)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, ctx.Err()
		}
		return false, nil
	case errors.Is(err, context.Canceled):
		return false, err
	}
	return true, err
}
