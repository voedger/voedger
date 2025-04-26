/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomLowercaseDigits_BasicUsage(t *testing.T) {
	log.Println(RandomLowercaseDigits(26))
	require.Len(t, RandomLowercaseDigits(26), 26)
	require.Empty(t, RandomLowercaseDigits(0))
	require.NotEqual(t, RandomLowercaseDigits(26), RandomLowercaseDigits((26)))
}
