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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, asp istructs.IAppStructsProvider, itokens itokens.ITokens,
	federation coreutils.IFederation, ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateLogin,
		execCmdCreateLogin(asp),
	))

	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(RegistryPackage, "IssuePrincipalToken"),
		provideIssuePrincipalTokenExec(asp, itokens)))
	provideChangePassword(cfg)
	provideResetPassword(cfg, asp, itokens, federation)
	cfg.AddAsyncProjectors(provideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation, cfg.Name, itokens))
	apps.RegisterSchemaFS(schemasFS, RegistryPackageFQN, ep)
}

func provideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameProjectorInvokeCreateWorkspaceID_registry,
			Func: invokeCreateWorkspaceIDProjector(federation, appQName, tokensAPI),
		}
	}
}

func ProvideSyncProjectorLoginIdxFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: QNameProjectorLoginIdx,
			Func: projectorLoginIdx,
		}
	}
}
