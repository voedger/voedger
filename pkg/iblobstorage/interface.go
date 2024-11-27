/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import (
	"context"
	"io"
)

type IBLOBKey interface {
	Bytes() []byte
}

type IBLOBStorage interface {
	// Errors: ErrBLOBSizeQuotaExceeded
	WriteBLOB(ctx context.Context, key PersistentBLOBKeyType, descr DescrType, reader io.Reader, limiter WLimiterType) (err error)

	// ttl is 2^duration hours
	// Errors: ErrBLOBSizeQuotaExceeded
	WriteTempBLOB(ctx context.Context, key TempBLOBKeyType, descr DescrType, reader io.Reader, limiter WLimiterType, Duration DurationType) (err error)

	// Function calls stateCallback then writer
	// Both stateCallback and writer can be nil
	// Errors: ErrBLOBNotFound, ErrBLOBCorrupted
	ReadBLOB(ctx context.Context, key IBLOBKey, stateCallback func(state BLOBState) error, writer io.Writer) (err error)

	// Wrapper around ReadBLOB() with nil writer argument
	QueryBLOBState(ctx context.Context, key IBLOBKey) (state BLOBState, err error)
}
