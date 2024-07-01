/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type uintType uint32

type intType int32

func TestUintToString(t *testing.T) {
	var n uint32 = 42
	require.Equal(t, "42", UintToString(n))

	var nTyped uintType = 43
	require.Equal(t, "43", UintToString(nTyped))
}

func TestIntToString(t *testing.T) {
	var n int32 = 42
	require.Equal(t, "42", IntToString(n))

	var nTyped intType = 43
	require.Equal(t, "43", IntToString(nTyped))
}
