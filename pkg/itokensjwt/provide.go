/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 *
 */

package itokensjwt

import (
	"sync"

	"github.com/golang-jwt/jwt"
	"github.com/voedger/voedger/pkg/coreutils"
	itokens "github.com/voedger/voedger/pkg/itokens"
)

// jwt.TimeFunc will be set to timeFunc but jwt.TimeFunc can not be protected from simulaneous access -> must be set only once
// e.g. to avoid data race: write jwt.TimeFunc in next test, read in async istructs.Projector of a previous test
var onceJWTTimeFuncSetter = sync.Once{}

var initial coreutils.ITime

// ProvideITokens implementation by provided interface
// To receive implementation you must provide Secret Key. Min length - 64 byte, panic otherwise
// panics on try to init again with different ITime implementation to avoid cases when we have >1 VIT that uses different times
// jwt.TimeFunc is single point of the time per the whole process so all tests must share exactly one implementation of ITime
func ProvideITokens(secretKey SecretKeyType, time coreutils.ITime) (tokenImpl itokens.ITokens) {
	onceJWTTimeFuncSetter.Do(func() {
		jwt.TimeFunc = time.Now
		initial = time
	})
	if initial != time {
		panic("tokens engine can not be initialized with different ITime implementation. Provide coreutils.MockTime only in tests and the singleton ITime implementation in runtime")
	}
	return NewJWTSigner(secretKey)
}
