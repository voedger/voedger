/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isequencer

import "time"

const (
	DefaultLRUCacheSize                      = 100_000
	DefaultMaxNumUnflushedValues             = 500
	defaultBatcherDelayOnToBeFlushedOverflow = 5 * time.Millisecond
)
