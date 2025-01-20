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

	// blobID for persistent, SUUID for temporary
	ID() string

	IsPersistent() bool
}

type IBLOBStorage interface {
	// Errors: ErrBLOBSizeQuotaExceeded
	WriteBLOB(ctx context.Context, key PersistentBLOBKeyType, descr DescrType, reader io.Reader, limiter WLimiterType) (err error)

	// blob TTL is 2^duration hours
	// Errors: ErrBLOBSizeQuotaExceeded
	WriteTempBLOB(ctx context.Context, key TempBLOBKeyType, descr DescrType, reader io.Reader, limiter WLimiterType, duration DurationType) (err error)

	// Function calls stateCallback then writer
	// stateCallback can be nil
	// Errors: ErrBLOBNotFound, ErrBLOBCorrupted
	ReadBLOB(ctx context.Context, key IBLOBKey, stateCallback func(state BLOBState) error, writer io.Writer, limiter RLimiterType) (err error)

	// state missing but the blob data exists -> ErrBLOBNotFound unlike ReadBLOB: it will return ErrBLOBCorrupted in this case
	QueryBLOBState(ctx context.Context, key IBLOBKey) (state BLOBState, err error)
}
