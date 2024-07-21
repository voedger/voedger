/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package authnz

import (
	"github.com/voedger/voedger/pkg/appdef"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder, itokens itokens.ITokens, atf payloads.IAppTokensFactory) {
	sprb.AddFunc(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "RefreshPrincipalToken"),
		provideRefreshPrincipalTokenExec(itokens),
	))
	sprb.AddFunc(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "EnrichPrincipalToken"),
		provideExecQryEnrichPrincipalToken(atf),
	))
}
