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
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys/authnz/signupin"
	"github.com/voedger/voedger/pkg/sys/authnz/workspace"
	"github.com/voedger/voedger/pkg/sys/authnz/wskinds"
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
	asp istructs.IAppStructsProvider, atf payloads.IAppTokensFactory, numCommandProcessors coreutils.CommandProcessorsCount, buildInfo *debug.BuildInfo, buildSubjectsIdx bool) {
	blobber.ProvideBlobberCmds(cfg, appDefBuilder)
	collection.ProvideCollectionFunc(cfg, appDefBuilder)
	collection.ProvideCDocFunc(cfg, appDefBuilder)
	collection.ProvideStateFunc(cfg, appDefBuilder)
	journal.Provide(cfg, appDefBuilder, ep)
	wskinds.ProvideCDocsWorkspaceKinds(appDefBuilder)
	builtin.Provide(cfg, appDefBuilder, buildInfo)
	builtin.ProvideQryEcho(cfg, appDefBuilder)
	builtin.ProvideQryGRCount(cfg, appDefBuilder)
	workspace.Provide(cfg, appDefBuilder, asp, timeFunc, itokens, federation)
	sqlquery.Provide(cfg, appDefBuilder, asp, numCommandProcessors)
	projectors.ProvideOffsetsDef(appDefBuilder)
	processors.ProvideJSONFuncParamsDef(appDefBuilder)
	verifier.Provide(cfg, appDefBuilder, itokens, federation, asp)
	signupin.ProvideQryRefreshPrincipalToken(cfg, appDefBuilder, itokens)
	signupin.ProvideCDocLogin(appDefBuilder)
	signupin.ProvideCmdEnrichPrincipalToken(cfg, appDefBuilder, atf)
	invite.Provide(cfg, appDefBuilder, timeFunc, buildSubjectsIdx)
	cfg.AddAsyncProjectors(
		journal.ProvideWLogDatesAsyncProjectorFactory(),
		workspace.ProvideAsyncProjectorFactoryInvokeCreateWorkspace(federation, cfg.Name, itokens),
		workspace.ProvideAsyncProjectorFactoryInvokeCreateWorkspaceID(federation, cfg.Name, itokens),
		workspace.ProvideAsyncProjectorInitializeWorkspace(federation, timeFunc, cfg.Name, ep, itokens, wsPostInitFunc),
		verifier.ProvideAsyncProjectorFactory_SendEmailVerificationCode(federation, smtpCfg),
		invite.ProvideAsyncProjectorApplyInvitationFactory(timeFunc, federation, cfg.Name, itokens, smtpCfg),
		invite.ProvideAsyncProjectorApplyJoinWorkspaceFactory(timeFunc, federation, cfg.Name, itokens),
		invite.ProvideAsyncProjectorApplyUpdateInviteRolesFactory(timeFunc, federation, cfg.Name, itokens, smtpCfg),
		invite.ProvideAsyncProjectorApplyCancelAcceptedInviteFactory(timeFunc, federation, cfg.Name, itokens),
		invite.ProvideAsyncProjectorApplyLeaveWorkspaceFactory(timeFunc, federation, cfg.Name, itokens),
	)
	cfg.AddSyncProjectors(
		workspace.ProvideSyncProjectorChildWorkspaceIdxFactory(),
		invite.ProvideSyncProjectorInviteIndexFactory(),
		invite.ProvideSyncProjectorJoinedWorkspaceIndexFactory(),
		workspace.ProvideAsyncProjectorWorkspaceIDIdx(),
	)
	cfg.AddSyncProjectors(collection.ProvideSyncProjectorFactories(appDefBuilder)...)
	uniques.Provide(cfg, appDefBuilder)
	describe.Provide(cfg, asp, appDefBuilder)

	// add sys sql schema
	sysSQLContent, err := sysFS.ReadFile("sys.sql")
	if err != nil {
		// notest
		panic(err)
	}
	sysFileScehmaAST, err := parser.ParseFile("sys.sql", string(sysSQLContent))
	if err != nil {
		// notest
		panic(err)
	}
	epFileSchemaASTs := ep.ExtensionPoint(apps.EPPackageSchemasASTs)
	epFileSchemaASTs.AddNamed(appdef.SysPackage, sysFileScehmaAST)
}
