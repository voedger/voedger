/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package payloads

import itokens "github.com/voedger/voedger/pkg/itokens"

var TestAppTokensFactory = func(tokens itokens.ITokens) IAppTokensFactory {
	return ProvideIAppTokensFactory(tokens)
}
