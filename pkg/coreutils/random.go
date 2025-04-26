/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import "crypto/rand"

const base32alphabet = "abcdefghijklmnopqrstuvwxyz234567"

func RandomLowercaseDigits(l int) string {
	src := make([]byte, l)
	rand.Read(src)
	for i := range src {
		src[i] = base32alphabet[src[i]%32]
	}
	return string(src)
}
