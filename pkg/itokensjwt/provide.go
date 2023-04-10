/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 *
 */

package itokensjwt

import (
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	itokens "github.com/untillpro/voedger/pkg/itokens"
)

// jwt.TimeFunc will be set to timeFunc but jwt.TimeFunc can not be protected from simulaneous access -> must be set only once
// e.g. to avoid data race in airs-bp3: write jwt.TimeFunc in next test, read in async istructs.Projector of a previous test
var onceJWTTimeFuncSetter = sync.Once{}

// ProvideITokens implementation by provided interface
// To receive implementation you must provide Secret Key. Min length - 64 byte, panic otherwise
func ProvideITokens(secretKey SecretKeyType, timeFunc func() time.Time) (tokenImpl itokens.ITokens) {
	onceJWTTimeFuncSetter.Do(func() {
		jwt.TimeFunc = timeFunc
	})
	return NewJWTSigner(secretKey)
}
