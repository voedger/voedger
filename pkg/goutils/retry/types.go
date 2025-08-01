/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @uathor Denis Gribanov
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier

import "time"

// Config holds parameters for retry behavior.
// ResetAfter defines a period after which the backoff interval resets.
type Config struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	JitterFactor    float64 // between 0 and 1
	ResetAfter      time.Duration
	// OnRetry is called before each retry with attempt number and next delay.
	OnRetry func(attempt int, delay time.Duration)
}

// Retrier executes operations with backoff, jitter, and reset logic.
type Retrier struct {
	cfg             Config
	currentInterval time.Duration
	lastReset       time.Time
}
