/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"

	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

func NewIN10NProc(vvmCtx context.Context, n10nBroker in10n.IN10nBroker, authenticator iauthnz.IAuthenticator,
	appTokensFactory payloads.IAppTokensFactory, asp istructs.IAppStructsProvider) (IN10NProc, func()) {
	res := &implIN10NProc{
		n10nBroker:         n10nBroker,
		authenticator:      authenticator,
		appTokensFactory:   appTokensFactory,
		appStructsProvider: asp,
	}
	return res, res.goroutinesWG.Wait
}
