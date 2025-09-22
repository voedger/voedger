/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	fmt.Println(HashBytes([]byte("hello world")))
	run1 := HashBytes([]byte("str1"))
	run2 := HashBytes([]byte("str1"))
	require.Equal(t, run1, run2)
	require.NotEqual(t, HashBytes([]byte("str1")), HashBytes([]byte("str2")))
}

func TestLoginHash(t *testing.T) {
	fmt.Println(LoginHash("hello world"))
	run1 := LoginHash("str1")
	run2 := LoginHash("str1")
	require.Equal(t, run1, run2)
	require.NotEqual(t, LoginHash("str1"), LoginHash("str2"))
}
