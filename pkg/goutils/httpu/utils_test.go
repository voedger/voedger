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

func TestServerAddress(t *testing.T) {
	require := require.New(t)

	t.Run("LocalhostAddress binds to localhost:0 only", func(t *testing.T) {
		require.Equal("127.0.0.1:0", LocalhostDynamic())
	})

	t.Run("ListenAddr returns localhost:0 for port 0", func(t *testing.T) {
		require.Equal("127.0.0.1:0", ListenAddr(0))
	})

	t.Run("ListenAddr returns public address for non-zero port", func(t *testing.T) {
		require.Equal(":8080", ListenAddr(8080))
	})
}
