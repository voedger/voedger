/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package authnz

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federation coreutils.IFederation,
	asp istructs.IAppStructsProvider, atf payloads.IAppTokensFactory) {
	cfgRegistry.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "RefreshPrincipalToken"),
		appdef.NullQName,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "RefreshPrincipalTokenResult")).
			AddField(field_NewPrincipalToken, appdef.DataKind_string, true).(appdef.IType).QName(),
		provideRefreshPrincipalTokenExec(itokens),
	))
	cfgRegistry.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "EnrichPrincipalToken"),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenParams")).
			AddField(Field_Login, appdef.DataKind_string, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenResult")).
			AddField(field_EnrichedToken, appdef.DataKind_string, true).(appdef.IType).QName(),
		provideExecQryEnrichPrincipalToken(atf),
	))
}
