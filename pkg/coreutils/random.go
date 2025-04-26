/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import "crypto/rand"

func DeviceRandomLoginPwd() (login, pwd string) {
	login = "device" + randomString(lowercaseDigitsAlphabet, deviceLoginAndPwdLen)
	pwd = randomString(lowercaseDigitsAlphabet, deviceLoginAndPwdLen)
	return login, pwd
}

func EmailVerificationCode() (verificationCode string) {
	return randomString(emailVerificationCodeAlphabet, emailVerificationCodeLength)
}

func randomString(alphabet string, l int) string {
	src := make([]byte, l)
	_, err := rand.Read(src)
	if err != nil {
		// notest
		panic(err)
	}
	for i := range src {
		src[i] = alphabet[int(src[i])%len(alphabet)]
	}
	return string(src)
}
