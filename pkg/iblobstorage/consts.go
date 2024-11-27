/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

const (
	SUUIDRandomPartLen = 16
)

const (
	DurationType_1Hour   DurationType = 1
	DurationType_32Hours DurationType = 5
)

const (
	blobPrefix_null blobPrefix = iota
	blobPrefix_persistent
	blobPrefix_temporary
)
