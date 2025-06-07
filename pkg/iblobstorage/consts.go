/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

const (
	DurationType_1Day  = DurationType(1)
	SUUIDRandomPartLen = 16
	secondsInDay       = 86400
)

const (
	blobPrefix_null blobPrefix = iota
	blobPrefix_persistent
	blobPrefix_temporary
)
