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
	InitialDelay time.Duration
	MaxDelay     time.Duration // 0 only allowed if Multiplier == 1
	Multiplier   float64
	JitterFactor float64 // between 0 and 1
	ResetAfter   time.Duration

	// OnError is called on any non-acceptable error
	OnError func(attempt int, delay time.Duration, err error)

	// retry on any error from RetryOnlyOn, abort on any other error
	// empty -> any error (except context cancellation) retried
	RetryOnlyOn []error

	// errors treated as success
	// retrier returns no error if an acceptable error occurs
	Acceptable []error
}

// Retrier executes operations with backoff, jitter, and reset logic.
type Retrier struct {
	cfg          Config
	currentDelay time.Duration
	lastReset    time.Time
}
