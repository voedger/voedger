/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package signupin

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/vvm"
)

func Provide(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federationURL vvm.FederationURLType, asp istructs.IAppStructsProvider) {

	// c.sys.CreateLogin
	provideCmdCreateLogin(cfgRegistry, appDefBuilder, asp)

	// istructs.Projector<S, LoginIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewLoginIdx, func(b appdef.IViewBuilder) {
		b.AddPartField(field_AppWSID, appdef.DataKind_int64)
		b.AddClustColumn(field_AppIDLoginHash, appdef.DataKind_string)
		b.AddValueField(field_CDocLoginID, appdef.DataKind_int64, true)
	})

	// q.sys.IssuePrincipalToken
	iptQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalToken")
	iptParamsQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalTokenParams")
	appDefBuilder.AddObject(iptParamsQName).
		AddField(authnz.Field_Login, appdef.DataKind_string, true).
		AddField(field_Passwrd, appdef.DataKind_string, true).
		AddField(Field_AppName, appdef.DataKind_string, true)

	iptResQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalTokenResult")
	iptResDef := appDefBuilder.AddObject(iptResQName)
	iptResDef.
		AddField(authnz.Field_PrincipalToken, appdef.DataKind_string, true).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, true).
		AddField(authnz.Field_WSError, appdef.DataKind_string, true)

	issuePrincipalTokenQry := istructsmem.NewQueryFunction(iptQName, iptParamsQName, iptResQName,
		provideIssuePrincipalTokenExec(asp, itokens))
	cfgRegistry.Resources.Add(issuePrincipalTokenQry)

	provideResetPassword(cfgRegistry, appDefBuilder, itokens, federationURL, asp)
	provideChangePassword(cfgRegistry, appDefBuilder)
}

func ProvideCmdEnrichPrincipalToken(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, atf payloads.IAppTokensFactory) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "EnrichPrincipalToken"),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenParams")).
			AddField(field_Login, appdef.DataKind_string, true).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenResult")).
			AddField(field_EnrichedToken, appdef.DataKind_string, true).(appdef.IDef).QName(),
		provideExecQryEnrichPrincipalToken(atf),
	))
}

// CDoc<Login> must be known in each target app. "unknown ownerQName scheme CDoc<Login>" on c.sys.CreatWorkspaceID otherwise
// has no ownerApp field because it is sys/registry always
func ProvideCDocLogin(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddCDoc(authnz.QNameCDocLogin).
		AddField(authnz.Field_ProfileClusterID, appdef.DataKind_int32, true).
		AddField(field_PwdHash, appdef.DataKind_bytes, true).
		AddField(Field_AppName, appdef.DataKind_string, true).
		AddField(authnz.Field_SubjectKind, appdef.DataKind_int32, false).
		AddField(authnz.Field_LoginHash, appdef.DataKind_string, true).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, false).     // to be written after workspace init
		AddField(authnz.Field_WSError, appdef.DataKind_string, false). // to be written after workspace init
		AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true)
}

func provideCmdCreateLogin(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		authnz.QNameCommandCreateLogin,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateLoginParams")).
			AddField(authnz.Field_Login, appdef.DataKind_string, true).
			AddField(Field_AppName, appdef.DataKind_string, true).
			AddField(authnz.Field_SubjectKind, appdef.DataKind_int32, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true).
			AddField(authnz.Field_ProfileClusterID, appdef.DataKind_int32, true).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateLoginUnloggedParams")).
			AddField(field_Password, appdef.DataKind_string, true).(appdef.IDef).QName(),
		appdef.NullQName,
		execCmdCreateLogin(asp),
	))
}

func ProvideSyncProjectorLoginIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameViewLoginIdx,
			Func: loginIdxProjector,
		}
	}
}
