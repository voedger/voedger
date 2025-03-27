/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 *
 */

package itokensjwt

import (
	"github.com/voedger/voedger/pkg/coreutils"
	itokens "github.com/voedger/voedger/pkg/itokens"
)

// ProvideITokens implementation by provided interface
// To receive implementation you must provide Secret Key. Min length - 64 byte, panic otherwise
// panics on try to init again with different ITime implementation to avoid cases when we have >1 VIT that uses different times
// jwt.TimeFunc is single point of the time per the whole process so all tests must share exactly one implementation of ITime
func ProvideITokens(secretKey SecretKeyType, time coreutils.ITime) (tokenImpl itokens.ITokens) {
	return NewJWTSigner(secretKey, time)
}
