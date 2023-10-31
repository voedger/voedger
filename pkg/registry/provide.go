/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider, itokens itokens.ITokens,
	federation coreutils.IFederation, ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateLogin,
		appdef.NullQName,
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateLogin(asp),
	))
	// istructs.Projector<S, LoginIdx>
	projectors.ProvideViewDef(appDefBuilder, QNameViewLoginIdx, func(b appdef.IViewBuilder) {
		b.KeyBuilder().PartKeyBuilder().AddField(field_AppWSID, appdef.DataKind_int64)
		b.KeyBuilder().ClustColsBuilder().AddField(field_AppIDLoginHash, appdef.DataKind_string)
		b.ValueBuilder().AddField(field_CDocLoginID, appdef.DataKind_int64, true)
	})
	// q.registry.IssuePrincipalToken
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(RegistryPackage, "IssuePrincipalToken"),
		appdef.NullQName,
		appdef.NullQName,
		provideIssuePrincipalTokenExec(asp, itokens)))
	provideChangePassword(cfg, appDefBuilder)
	provideResetPassword(cfg, appDefBuilder, asp, itokens, federation)

	apps.RegisterSchemaFS(schemasFS, RegistryPackage, ep)
}

func ProvideSyncProjectorLoginIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorLoginIdx,
			Func: projectorLoginIdx,
		}
	}
}
