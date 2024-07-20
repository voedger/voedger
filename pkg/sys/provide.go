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
	"github.com/voedger/voedger/pkg/utils/federation"
)

//go:embed *.vsql
var SysFS embed.FS

func ProvideStateless(spb istructsmem.IStatelessPkgBuilder) {
	sprb := spb.AddPackage(appdef.SysPackage)
	blobber.ProvideBlobberCmds(sprb)
}

func Provide(cfg *istructsmem.AppConfigType, smtpCfg smtp.Cfg,
	ep extensionpoints.IExtensionPoint, wsPostInitFunc workspace.WSPostInitFunc, timeFunc coreutils.TimeFunc, itokens itokens.ITokens, federation federation.IFederation,
	asp istructs.IAppStructsProvider, atf payloads.IAppTokensFactory, buildInfo *debug.BuildInfo,
	storageProvider istorage.IAppStorageProvider) parser.PackageFS {
	// blobber.ProvideBlobberCmds(cfg)
	collection.Provide(cfg)
	journal.Provide(cfg, ep)
	builtin.Provide(cfg, buildInfo, storageProvider)
	workspace.Provide(cfg, timeFunc, itokens, federation, itokens, ep, wsPostInitFunc)
	sqlquery.Provide(cfg, asp)
	projectors.ProvideOffsetsDef(cfg.AppDefBuilder())
	verifier.Provide(cfg, itokens, federation, asp, smtpCfg, timeFunc)
	authnz.Provide(cfg, itokens, atf)
	invite.Provide(cfg, timeFunc, federation, itokens, smtpCfg)
	uniques.Provide(cfg)
	describe.Provide(cfg)
	return parser.PackageFS{
		Path: appdef.SysPackage,
		FS:   SysFS,
	}
}
