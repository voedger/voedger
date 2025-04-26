/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandom(t *testing.T) {
	dl, dp := DeviceRandomLoginPwd()
	log.Println(dl, dp)
	require.True(t, strings.HasPrefix(dl, "device"))
	require.Len(t, dp, deviceLoginAndPwdLen)
	require.Len(t, dl, deviceLoginAndPwdLen+6)
	dl1, dp1 := DeviceRandomLoginPwd()
	require.NotEqual(t, dl, dl1)
	require.NotEqual(t, dp, dp1)

	evc := EmailVerificationCode()
	log.Println(evc)
	evc1 := EmailVerificationCode()
	require.NotEqual(t, evc, evc1)
}
