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

const (
	// no trust at all, InsertIfNotExists only
	SequencesTrustLevel_0 SequencesTrustLevel = iota

	// no trust to log writes, trust to records
	SequencesTrustLevel_1

	// trust to everything
	SequencesTrustLevel_2
)
