/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys

import (
	"embed"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/blobber"
	"github.com/voedger/voedger/pkg/sys/builtin"
	"github.com/voedger/voedger/pkg/sys/collection"
	"github.com/voedger/voedger/pkg/sys/describe"
	"github.com/voedger/voedger/pkg/sys/invite"
	"github.com/voedger/voedger/pkg/sys/journal"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sqlquery"
	"github.com/voedger/voedger/pkg/sys/uniques"
	"github.com/voedger/voedger/pkg/sys/verifier"
	"github.com/voedger/voedger/pkg/sys/workspace"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

//go:embed sys.sql
var SysFS embed.FS

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, smtpCfg smtp.Cfg,
	ep extensionpoints.IExtensionPoint, wsPostInitFunc workspace.WSPostInitFunc, timeFunc coreutils.TimeFunc, itokens itokens.ITokens, federation coreutils.IFederation,
	asp istructs.IAppStructsProvider, atf payloads.IAppTokensFactory, numCommandProcessors coreutils.CommandProcessorsCount, buildInfo *debug.BuildInfo,
	storageProvider istorage.IAppStorageProvider) parser.PackageFS {
	blobber.ProvideBlobberCmds(cfg)
	collection.Provide(cfg, appDefBuilder)
	journal.Provide(cfg, ep)
	builtin.Provide(cfg, appDefBuilder, buildInfo, storageProvider)
	workspace.Provide(cfg, appDefBuilder, asp, timeFunc, itokens, federation, itokens, ep, wsPostInitFunc)
	sqlquery.Provide(cfg, asp, numCommandProcessors)
	projectors.ProvideOffsetsDef(appDefBuilder)
	verifier.Provide(cfg, itokens, federation, asp, smtpCfg, timeFunc)
	authnz.Provide(cfg, itokens, atf)
	invite.Provide(cfg, timeFunc, federation, itokens, smtpCfg)
	uniques.Provide(cfg, appDefBuilder)
	describe.Provide(cfg, asp)
	return ProvidePackageFS()
}

func ProvidePackageFS() parser.PackageFS {
	return parser.PackageFS{
		QualifiedPackageName: appdef.SysPackage,
		FS:                   SysFS,
	}
}
