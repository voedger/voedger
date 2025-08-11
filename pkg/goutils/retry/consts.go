/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @author Denis Gribanov
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier

const (
	DefaultJitterFactor = 0.5
	DefaultMultiplier   = 2
)

const (
	DoRetry Action = iota

	// consider the current error as an acceptable result, return nil
	Accept

	// consider the further retries as senceless, return the current error
	Abort
)
