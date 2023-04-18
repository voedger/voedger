/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package payloads

import itokens "github.com/voedger/voedger/pkg/itokens"

func ProvideIAppTokensFactory(tokens itokens.ITokens) IAppTokensFactory {
	return &implIAppTokensFactory{
		tokens: tokens,
	}
}
