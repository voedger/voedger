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

	// HandleError is called on any non-acceptable error
	HandleError func(attempt int, delay time.Duration, err error) Action
}

type Action int

// Retrier executes operations with backoff, jitter, and reset logic.
type Retrier struct {
	cfg          Config
	currentDelay time.Duration
	lastReset    time.Time
}
