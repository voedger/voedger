/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package signupin

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federation coreutils.IFederation,
	asp istructs.IAppStructsProvider, ep extensionpoints.IExtensionPoint) {

	// c.sys.CreateLogin
	provideCmdCreateLogin(cfgRegistry, appDefBuilder, asp)

	// istructs.Projector<S, LoginIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewLoginIdx, func(b appdef.IViewBuilder) {
		b.KeyBuilder().PartKeyBuilder().AddField(field_AppWSID, appdef.DataKind_int64)
		b.KeyBuilder().ClustColsBuilder().AddStringField(field_AppIDLoginHash, appdef.DefaultFieldMaxLength)
		b.ValueBuilder().AddField(field_CDocLoginID, appdef.DataKind_int64, true)
	})

	// q.sys.IssuePrincipalToken
	iptQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalToken")
	iptParamsQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalTokenParams")
	appDefBuilder.AddObject(iptParamsQName).
		AddField(authnz.Field_Login, appdef.DataKind_string, true).
		AddField(field_Passwrd, appdef.DataKind_string, true).
		AddField(Field_AppName, appdef.DataKind_string, true)

	iptResQName := appdef.NewQName(appdef.SysPackage, "IssuePrincipalTokenResult")
	iptRes := appDefBuilder.AddObject(iptResQName)
	iptRes.
		AddField(authnz.Field_PrincipalToken, appdef.DataKind_string, true).
		AddField(authnz.Field_WSID, appdef.DataKind_int64, true).
		AddField(authnz.Field_WSError, appdef.DataKind_string, true)

	issuePrincipalTokenQry := istructsmem.NewQueryFunction(iptQName, iptParamsQName, iptResQName,
		provideIssuePrincipalTokenExec(asp, itokens))
	cfgRegistry.Resources.Add(issuePrincipalTokenQry)

	provideResetPassword(cfgRegistry, appDefBuilder, asp, itokens, federation)
	provideChangePassword(cfgRegistry, appDefBuilder)

}

func ProvideCmdEnrichPrincipalToken(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, atf payloads.IAppTokensFactory) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "EnrichPrincipalToken"),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenParams")).
			AddField(field_Login, appdef.DataKind_string, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EnrichPrincipalTokenResult")).
			AddField(field_EnrichedToken, appdef.DataKind_string, true).(appdef.IType).QName(),
		provideExecQryEnrichPrincipalToken(atf),
	))
}

// CDoc<Login> must be known in each target app. "unknown ownerQName scheme CDoc<Login>" on c.sys.CreatWorkspaceID otherwise
// has no ownerApp field because it is sys/registry always
func ProvideCDocLogin(appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	apps.Parse(schemasFS, appdef.SysPackage, ep)
}

func provideCmdCreateLogin(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		authnz.QNameCommandCreateLogin,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateLoginParams")).
			AddField(authnz.Field_Login, appdef.DataKind_string, true).
			AddField(Field_AppName, appdef.DataKind_string, true).
			AddField(authnz.Field_SubjectKind, appdef.DataKind_int32, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true).
			AddField(authnz.Field_ProfileClusterID, appdef.DataKind_int32, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateLoginUnloggedParams")).
			AddField(field_Password, appdef.DataKind_string, true).(appdef.IType).QName(),
		appdef.NullQName,
		execCmdCreateLogin(asp),
	))
}

func ProvideSyncProjectorLoginIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorLoginIdx,
			Func: projectorLoginIdx,
		}
	}
}
