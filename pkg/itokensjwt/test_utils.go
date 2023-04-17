/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package itokensjwt

import (
	itokens "github.com/voedger/voedger/pkg/itokens"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

var TestTokensJWT = func() itokens.ITokens {
	return ProvideITokens(SecretKeyExample, coreutils.TestTimeFunc)
}
