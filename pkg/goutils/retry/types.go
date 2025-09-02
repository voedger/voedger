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
	BaseDelay               time.Duration // required
	MaxDelay                time.Duration // required
	OnError                 func(attempt int, delay time.Duration, opErr error) (retry bool, abortErr error)
	ResetDelayAfterMaxDelay bool
}

// Retrier executes operations with backoff, jitter, and reset logic.
type Retrier struct {
	cfg     Config
	attempt int
}
