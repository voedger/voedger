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
func ProvideITokens(secretKey SecretKeyType, time coreutils.ITime) (tokenImpl itokens.ITokens) {
	return NewJWTSigner(secretKey, time)
}
