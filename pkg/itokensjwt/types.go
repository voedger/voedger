/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package itokensjwt

type SecretKeyType []byte

type JWTSigner struct {
	secretKey []byte
}
