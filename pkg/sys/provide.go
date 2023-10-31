/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys

import (
	"embed"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/workspace"
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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

//go:embed sys.sql
var sysFS embed.FS

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, smtpCfg smtp.Cfg,
	ep extensionpoints.IExtensionPoint, wsPostInitFunc workspace.WSPostInitFunc, timeFunc coreutils.TimeFunc, itokens itokens.ITokens, federation coreutils.IFederation,
	asp istructs.IAppStructsProvider, atf payloads.IAppTokensFactory, numCommandProcessors coreutils.CommandProcessorsCount, buildInfo *debug.BuildInfo,
	storageProvider istorage.IAppStorageProvider, rebuildRegistry bool) {
	blobber.ProvideBlobberCmds(cfg)
	collection.Provide(cfg, appDefBuilder)
	journal.Provide(cfg, appDefBuilder, ep)
	builtin.Provide(cfg, appDefBuilder, buildInfo, storageProvider, rebuildRegistry)
	workspace.Provide(cfg, appDefBuilder, asp, timeFunc, itokens, federation, itokens, ep, wsPostInitFunc)
	sqlquery.Provide(cfg, appDefBuilder, asp, numCommandProcessors)
	projectors.ProvideOffsetsDef(appDefBuilder)
	processors.ProvideJSONFuncParamsDef(appDefBuilder)
	verifier.Provide(cfg, appDefBuilder, itokens, federation, asp, smtpCfg, timeFunc)
	authnz.Provide(cfg, appDefBuilder, itokens, federation, asp, atf)
	invite.Provide(cfg, appDefBuilder, timeFunc, federation, itokens, smtpCfg)
	uniques.Provide(cfg, appDefBuilder)
	describe.Provide(cfg, asp, appDefBuilder)

	// add sys sql schema
	apps.RegisterSchemaFS(sysFS, appdef.SysPackage, ep)
}
