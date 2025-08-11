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
	InitialDelay time.Duration
	MaxDelay     time.Duration // >=0. 0 - not used. 0 only allowed if Multiplier == 1
	Multiplier   float64       // >1
	JitterFactor float64       // between 0 and 1
	ResetAfter   time.Duration
	OnError      func(attempt int, delay time.Duration, err error) Action
}

// retry policy: DoRetry, Abort, Accept
type Action int

// Retrier executes operations with backoff, jitter, and reset logic.
type Retrier struct {
	cfg          Config
	currentDelay time.Duration
	lastReset    time.Time
}
