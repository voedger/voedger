/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import "crypto/rand"

// generates cryptographically secure random verification code. Len - 6 bytes, each value is 0-9
func EmailVerificationCode() (verificationCode string, err error) {
	verificationCodeBytes := make([]byte, emailVerificationCodeLength)
	if _, err := rand.Read(verificationCodeBytes); err != nil {
		// notest
		return "", err
	}

	// compress range 0..255 -> 0..9
	for i := 0; i < len(verificationCodeBytes); i++ {
		verificationCodeBytes[i] = emailVerificationCodeSymbols[int(float32(verificationCodeBytes[i])/byteRangeToEmailVerifcationSymbolsRangeCoeff)]
	}
	return string(verificationCodeBytes), nil
}
