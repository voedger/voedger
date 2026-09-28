/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isequencer

const (
	// no trust at all, InsertIfNotExists only
	SequencesTrustLevel_0 SequencesTrustLevel = iota

	// no trust to log writes, trust to records
	SequencesTrustLevel_1

	// trust to everything
	SequencesTrustLevel_2
)
