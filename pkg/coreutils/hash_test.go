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
	require.Equal(t, HashBytes([]byte("str1")), HashBytes([]byte("str1")))
	require.NotEqual(t, HashBytes([]byte("str1")), HashBytes([]byte("str2")))
}
