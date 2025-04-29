/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registry

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/parser"
	_ "github.com/voedger/voedger/pkg/sys"
)

func Provide(cfg *istructsmem.AppConfigType, itokens itokens.ITokens, federation federation.IFederation) parser.PackageFS {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateLogin,
		execCmdCreateLogin,
	))
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandCreateEmailLogin,
		execCmdCreateEmailLogin,
	))

	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(RegistryPackage, "IssuePrincipalToken"),
		provideIssuePrincipalTokenExec(itokens)))
	provideChangePassword(cfg)
	provideResetPassword(cfg, itokens, federation)
	cfg.AddAsyncProjectors(
		provideAsyncProjectorInvokeCreateWorkspaceID(federation, itokens),
	)
	return ProvidePackageFS()
}

func ProvidePackageFS() parser.PackageFS {
	return parser.PackageFS{
		Path: RegistryPackageFQN,
		FS:   schemasFS,
	}
}

func provideAsyncProjectorInvokeCreateWorkspaceID(federation federation.IFederation, tokensAPI itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameProjectorInvokeCreateWorkspaceID_registry,
		Func: invokeCreateWorkspaceIDProjector(federation, tokensAPI),
	}
}

func ProvideSyncProjectorLoginIdx() istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorLoginIdx,
		Func: projectorLoginIdx,
	}
}
