/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registry

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/parser"
	_ "github.com/voedger/voedger/pkg/sys"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, asp istructs.IAppStructsProvider, itokens itokens.ITokens,
	federation coreutils.IFederation) parser.PackageFS {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateLogin,
		execCmdCreateLogin(asp),
	))

	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(RegistryPackage, "IssuePrincipalToken"),
		provideIssuePrincipalTokenExec(itokens)))
	provideChangePassword(cfg)
	provideResetPassword(cfg, itokens, federation)
	cfg.AddAsyncProjectors(
		provideAsyncProjectorInvokeCreateWorkspaceID(federation, cfg.Name, itokens),
	)
	return ProvidePackageFS()
}

func ProvidePackageFS() parser.PackageFS {
	return parser.PackageFS{
		Path: RegistryPackageFQN,
		FS:   schemasFS,
	}
}

func provideAsyncProjectorInvokeCreateWorkspaceID(federation coreutils.IFederation, appQName istructs.AppQName, tokensAPI itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameProjectorInvokeCreateWorkspaceID_registry,
		Func: invokeCreateWorkspaceIDProjector(federation, appQName, tokensAPI),
	}
}

func ProvideSyncProjectorLoginIdx() istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorLoginIdx,
		Func: projectorLoginIdx,
	}
}
