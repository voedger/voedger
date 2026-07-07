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

	// Required by vpm/pkg/compile: appws.vsql references `sys` package,
	// so pkg/sys golang package must appear in the Go import graph for packages.Load
	// to discover its directory and embedded *.vsql.
	// The import exists purely as a build-time signal for the vsql loader.
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
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandInitiateSetLoginAlias,
		execCmdInitiateSetLoginAlias,
	))
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandPutLoginAliasIndex,
		execCmdPutLoginAliasIndex,
	))
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandDeactivateLoginAliasIndex,
		execCmdDeactivateLoginAliasIndex,
	))

	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(RegistryPackage, "IssuePrincipalToken"),
		provideIssuePrincipalTokenExec(itokens, federation)))
	provideChangePassword(cfg)
	provideResetPassword(cfg, itokens, federation)
	provideUpdateGlobalRoles(cfg)
	cfg.AddAsyncProjectors(
		provideAsyncProjectorInvokeCreateWorkspaceID(federation.WithRetry(), itokens),
		provideAsyncProjectorApplySetLoginAlias(federation.WithRetry(), itokens),
	)
	return ProvidePackageFS()
}

func ProvidePackageFS() parser.PackageFS {
	return parser.PackageFS{
		Path: RegistryPackageFQN,
		FS:   schemasFS,
	}
}

func provideAsyncProjectorInvokeCreateWorkspaceID(federation federation.IFederationWithRetry, tokensAPI itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameProjectorInvokeCreateWorkspaceID_registry,
		Func: invokeCreateWorkspaceIDProjector(federation, tokensAPI),
	}
}

func provideAsyncProjectorApplySetLoginAlias(federation federation.IFederationWithRetry, tokensAPI itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorApplySetLoginAlias,
		Func: applySetLoginAlias(federation, tokensAPI),
	}
}

func ProvideSyncProjectorLoginIdx() istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorLoginIdx,
		Func: projectorLoginIdx,
	}
}
