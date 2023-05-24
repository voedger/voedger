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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federation coreutils.IFederation, asp istructs.IAppStructsProvider) {

	// c.sys.CreateLogin
	provideCmdCreateLogin(cfgRegistry, appDefBuilder, asp)

	// istructs.Projector<S, LoginIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewLoginIdx, func(b appdef.IViewBuilder) {
		b.PartKeyDef().AddField(field_AppWSID, appdef.DataKind_int64, true)
		b.ClustColsDef().AddField(field_AppIDLoginHash, appdef.DataKind_string, true)
		b.ValueDef().AddField(field_CDocLoginID, appdef.DataKind_int64, true)
	})

	// q.sys.IssuePrincipalToken
	iptQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalToken")
	iptParamsQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalTokenParams")
	appDefBuilder.AddStruct(iptParamsQName, appdef.DefKind_Object).
		AddField(authnz.Field_Login, appdef.DataKind_string, true).
		AddField(field_Passwrd, appdef.DataKind_string, true).
		AddField(Field_AppName, appdef.DataKind_string, true)

	iptResQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalTokenResult")
	iptResDef := appDefBuilder.AddStruct(iptResQName, appdef.DefKind_Object)
	iptResDef.
		AddField(authnz.Field_PrincipalToken, appdef.DataKind_string, true).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, true).
		AddField(authnz.Field_WSError, appdef.DataKind_string, true)

	issuePrincipalTokenQry := istructsmem.NewQueryFunction(iptQName, iptParamsQName, iptResQName,
		provideIssuePrincipalTokenExec(asp, itokens))
	cfgRegistry.Resources.Add(issuePrincipalTokenQry)

	provideResetPassword(cfgRegistry, appDefBuilder, itokens, federation, asp)
	provideChangePassword(cfgRegistry, appDefBuilder)
}

func ProvideCmdEnrichPrincipalToken(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, atf payloads.IAppTokensFactory) {

	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "EnrichPrincipalToken"),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenParams"), appdef.DefKind_Object).
			AddField(field_Login, appdef.DataKind_string, true).
			QName(),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenResult"), appdef.DefKind_Object).
			AddField(field_EnrichedToken, appdef.DataKind_string, true).
			QName(),
		provideExecQryEnrichPrincipalToken(atf),
	))
}

// CDoc<Login> must be known in each target app. "unknown ownerQName scheme CDoc<Login>" on c.sys.CreatWorkspaceID otherwise
// has no ownerApp field because it is sys/registry always
func ProvideCDocLogin(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddStruct(authnz.QNameCDocLogin, appdef.DefKind_CDoc).
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

		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "CreateLoginParams"), appdef.DefKind_Object).
			AddField(authnz.Field_Login, appdef.DataKind_string, true).
			AddField(Field_AppName, appdef.DataKind_string, true).
			AddField(authnz.Field_SubjectKind, appdef.DataKind_int32, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true).
			AddField(authnz.Field_ProfileClusterID, appdef.DataKind_int32, true).QName(),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "CreateLoginUnloggedParams"), appdef.DefKind_Object).
			AddField(field_Password, appdef.DataKind_string, true).QName(),
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
