/*
 * Copyright (c) 2020-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package registry

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider, itokens itokens.ITokens,
	federation coreutils.IFederation, ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateLogin,
		appDefBuilder.AddObject(appdef.NewQName(RegistryPackage, "CreateLoginParams")).
			AddField(authnz.Field_Login, appdef.DataKind_string, true).
			AddField(authnz.Field_AppName, appdef.DataKind_string, true).
			AddField(authnz.Field_SubjectKind, appdef.DataKind_int32, true).
			AddField(authnz.Field_WSKindInitializationData, appdef.DataKind_string, true).
			AddField(authnz.Field_ProfileClusterID, appdef.DataKind_int32, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(RegistryPackage, "CreateLoginUnloggedParams")).
			AddField(field_Password, appdef.DataKind_string, true).(appdef.IType).QName(),
		appdef.NullQName,
		execCmdCreateLogin(asp),
	))
	// istructs.Projector<S, LoginIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewLoginIdx, func(b appdef.IViewBuilder) {
		b.KeyBuilder().PartKeyBuilder().AddField(field_AppWSID, appdef.DataKind_int64)
		b.KeyBuilder().ClustColsBuilder().AddStringField(field_AppIDLoginHash, appdef.DefaultFieldMaxLength)
		b.ValueBuilder().AddField(field_CDocLoginID, appdef.DataKind_int64, true)
	})
	// q.sys.IssuePrincipalToken
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(RegistryPackage, "IssuePrincipalToken"),
		appDefBuilder.AddObject(appdef.NewQName(RegistryPackage, "IssuePrincipalTokenParams")).
			AddField(authnz.Field_Login, appdef.DataKind_string, true).
			AddField(field_Passwrd, appdef.DataKind_string, true).
			AddField(authnz.Field_AppName, appdef.DataKind_string, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(RegistryPackage, "IssuePrincipalTokenResult")).
			AddField(authnz.Field_PrincipalToken, appdef.DataKind_string, true).
			AddField(authnz.Field_WSID, appdef.DataKind_int64, true).
			AddField(authnz.Field_WSError, appdef.DataKind_string, true).(appdef.IType).QName(),
		provideIssuePrincipalTokenExec(asp, itokens)))
	provideChangePassword(cfg, appDefBuilder)
	provideResetPassword(cfg, appDefBuilder, asp, itokens, federation)
	apps.Parse(schemasFS, "registry", ep)
}

func ProvideSyncProjectorLoginIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorLoginIdx,
			Func: projectorLoginIdx,
		}
	}
}
