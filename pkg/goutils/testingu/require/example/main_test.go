/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package main

import (
	"errors"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestGoCrazy(t *testing.T) {
	require := require.New(t)

	require.Panics(
		GoCrazy,
		"GoCrazy should panics!",
		require.Is(ErrCrazyError, "panic error should be %v", ErrCrazyError),
		require.Is(errors.ErrUnsupported),
		require.Has("🤪", "panic should contains crazy smile %q", "🤪"),
		require.Has("unsupported"),
		require.NotHas("toxic"),
		require.Rx(`^.*\s+error`, "panic should contain `error` word"),
		require.NotRx(`^Santa`, "panic should starts from `Santa` word"),
	)
}

func TestCrazyError(t *testing.T) {
	require := require.New(t)

	require.Error(
		ErrCrazyError,
		"ErrCrazyError should be an error",
		require.Is(errors.ErrUnsupported, "error should be %v", errors.ErrUnsupported),
		require.Has("🤪"),
	)
}
