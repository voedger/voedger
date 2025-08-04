/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @author Denis Gribanov
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier

import "time"

// Config holds parameters for retry behavior and error-handling policies.
type Config struct {
	// Backoff settings
	InitialInterval time.Duration
	MaxInterval     time.Duration // 0 only allowed if Multiplier == 1
	Multiplier      float64
	JitterFactor    float64 // between 0 and 1
	ResetAfter      time.Duration

	// OnRetry is called before each retry with attempt number and next delay.
	OnRetry func(attempt int, delay time.Duration, err error)

	// errors that should trigger a retry.
	// if empty, all errors (except context cancellation) are retried.
	// not empty, abort on any other error
	RetryOn []error

	// errors treated as success
	Acceptable []error
}

// Retrier executes operations with backoff, jitter, and reset logic.
type Retrier struct {
	cfg             Config
	currentInterval time.Duration
	lastReset       time.Time
}
