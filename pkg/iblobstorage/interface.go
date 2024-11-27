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
	WriteBLOB(ctx context.Context, key IBLOBKey, descr DescrType, reader io.Reader, quoter WQuoterType) (err error)

	// Function calls stateCallback then writer
	// Both stackeCallback and writer can be nil
	// Errors: ErrBLOBNotFound
	ReadBLOB(ctx context.Context, key IBLOBKey, stateCallback func(state BLOBState) error, writer io.Writer) (err error)

	// Wrapper around ReadBLOB() with nil writer argument
	QueryBLOBState(ctx context.Context, key IBLOBKey) (state BLOBState, err error)

	// WriteTempBLOB(ctx context.Context, key TempBLOBKeyType, descr DescrType, reader io.Reader, duration DurationType, quoter WQuoterType) error
	// ReadTempBLOB(ctx context.Context, key TempBLOBKeyType, stateWriter func(state BLOBState) error, writer io.Writer, quoter RQuoterType) (err error)
}
