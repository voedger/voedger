/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package itokensjwt

import "github.com/voedger/voedger/pkg/coreutils"

type SecretKeyType []byte

type JWTSigner struct {
	secretKey []byte
	iTime coreutils.ITime
}
