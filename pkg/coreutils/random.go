/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import "crypto/rand"

const base32alphabet = "abcdefghijklmnopqrstuvwxyz234567"

func RandomLowercaseDigits(l int) string {
	src := make([]byte, l)
	_, err := rand.Read(src)
	if err != nil {
		// notest
		panic(err)
	}
	for i := range src {
		src[i] = base32alphabet[int(src[i])%len(base32alphabet)]
	}
	return string(src)
}
