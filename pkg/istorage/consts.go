/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istorage

const (
	AppStorageStatus_Pending AppStorageStatus = iota
	AppStorageStatus_Done
)

const (
	MaxSafeNameLength = 48 - 5 // max Cassandra keypsace name len - 5 symbols for prefix

	// failed to get unique name often than this value -> ErrAppQNameIsTooUnsafe
	maxMatchedOccurances = 3
)

var (
	SysMetaSafeName = SafeAppName{name: "sysmeta"}
)

