/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import (
	"context"
	"io"
)

//	"context"

type IBLOBKey interface {
	Bytes() []byte
}

type IBLOBStorage interface {
	// Errors: ErrBLOBSizeQuotaExceeded
	WriteBLOB(ctx context.Context, key KeyType, descr DescrType, reader io.Reader, maxSize BLOBMaxSizeType) (err error)

	// Function calls stateWriter then writer
	// Both stateWriter and writer can be nil
	// Errors: ErrBLOBNotFound
	ReadBLOB(ctx context.Context, key KeyType, stateWriter func(state BLOBState) error, writer io.Writer) (err error)

	// Wrapper around ReadBLOB() with nil writer argument
	QueryBLOBState(ctx context.Context, key IBLOBKey) (state BLOBState, err error)

	WriteTempBLOB(ctx context.Context, key TempKeyType, descr DescrType, reader io.Reader, duration DurationType, quoter WQuoterType) error
}
