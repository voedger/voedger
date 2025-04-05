/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isequencer

import "time"

const (
	DefaultLRUCacheSize          = 100_000
	DefaultMaxNumUnflushedValues = 500
	defaultRetryDelay            = 500 * time.Millisecond
	defaultRetryCount            = 0
)
