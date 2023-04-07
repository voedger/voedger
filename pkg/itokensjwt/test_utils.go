/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package itokensjwt

import (
	itokens "github.com/untillpro/voedger/pkg/itokens"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

var TestTokensJWT = func() itokens.ITokens {
	return ProvideITokens(SecretKeyExample, coreutils.TestTimeFunc)
}
