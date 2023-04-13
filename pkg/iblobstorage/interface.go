/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import (
	"context"
	"io"
)

//	"context"

type IBLOBStorage interface {
	// Errors: ErrBLOBSizeQuotaExceeded
	WriteBLOB(ctx context.Context, key KeyType, descr DescrType, reader io.Reader, maxSize int64) (err error)

	// Function calls stateWriter then writer
	// Both stateWriter and writer can be nil
	// Errors: ErrBLOBNotFound
	ReadBLOB(ctx context.Context, key KeyType, stateWriter func(state BLOBState) error, writer io.Writer) (err error)

	// Wrapper around ReadBLOB() with nil writer argument
	QueryBLOBState(ctx context.Context, key KeyType) (state BLOBState, err error)
}
