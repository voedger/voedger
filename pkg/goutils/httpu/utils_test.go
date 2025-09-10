/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWSAEError(t *testing.T) {
	require := require.New(t)
	err := &os.SyscallError{Err: syscall.Errno(123)}
	require.True(IsWSAEError(err, 123))
	require.False(IsWSAEError(err, 124))
	require.False(IsWSAEError(errors.New("x"), 123))
}
