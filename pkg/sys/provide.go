/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
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
	"github.com/voedger/voedger/pkg/vvm"
)

func Provide(timeFunc func() time.Time, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, vvmAPI vvm.VVMAPI, smtpCfg smtp.Cfg,
	sep vvm.IStandardExtensionPoints, wsPostInitFunc workspace.WSPostInitFunc) {
	blobber.ProvideBlobberCmds(cfg, appDefBuilder)
	collection.ProvideCollectionFunc(cfg, appDefBuilder)
	collection.ProvideCDocFunc(cfg, appDefBuilder)
	collection.ProvideStateFunc(cfg, appDefBuilder)
	journal.Provide(cfg, appDefBuilder, sep.EPJournalIndices(), sep.EPJournalPredicates())
	wskinds.ProvideCDocsWorkspaceKinds(appDefBuilder)
	builtin.ProvideCmdCUD(cfg)
	builtin.ProvideCmdInit(cfg)   // for import from air-importbo
	builtin.ProivdeCmdImport(cfg) // for sync
	builtin.ProvideQryEcho(cfg, appDefBuilder)
	builtin.ProvideQryGRCount(cfg, appDefBuilder)
	workspace.Provide(cfg, appDefBuilder, vvmAPI.IAppStructsProvider, timeFunc)
	sqlquery.Provide(cfg, appDefBuilder, vvmAPI.IAppStructsProvider, vvm.NumCommandProcessors)
	projectors.ProvideOffsetsDef(appDefBuilder)
	commandprocessor.ProvideJSONFuncParamsDef(appDefBuilder)
	verifier.Provide(cfg, appDefBuilder, vvmAPI.ITokens, vvmAPI.FederationURL, vvmAPI.IAppStructsProvider)
	signupin.ProvideQryRefreshPrincipalToken(cfg, appDefBuilder, vvmAPI.ITokens)
	signupin.ProvideCDocLogin(appDefBuilder)
	invite.Provide(cfg, appDefBuilder, timeFunc)
	cfg.AddAsyncProjectors(
		journal.ProvideWLogDatesAsyncProjectorFactory(),
		workspace.ProvideAsyncProjectorFactoryInvokeCreateWorkspace(vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens),
		workspace.ProvideAsyncProjectorFactoryInvokeCreateWorkspaceID(vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens),
		workspace.ProvideAsyncProjectorInitializeWorkspace(vvmAPI.FederationURL, timeFunc, cfg.Name, sep.EPWSTemplates(), vvmAPI.ITokens, wsPostInitFunc),
		verifier.ProvideAsyncProjectorFactory_SendEmailVerificationCode(vvmAPI.FederationURL, smtpCfg),
		invite.ProvideAsyncProjectorApplyInvitationFactory(timeFunc, vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens, smtpCfg),
		invite.ProvideAsyncProjectorApplyJoinWorkspaceFactory(timeFunc, vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens),
		invite.ProvideAsyncProjectorApplyUpdateInviteRolesFactory(timeFunc, vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens, smtpCfg),
		invite.ProvideAsyncProjectorApplyCancelAcceptedInviteFactory(timeFunc, vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens),
		invite.ProvideAsyncProjectorApplyLeaveWorkspaceFactory(timeFunc, vvmAPI.FederationURL, cfg.Name, vvmAPI.ITokens),
	)
	cfg.AddSyncProjectors(
		workspace.ProvideSyncProjectorChildWorkspaceIdxFactory(),
		invite.ProvideSyncProjectorInviteIndexFactory(),
		invite.ProvideSyncProjectorJoinedWorkspaceIndexFactory(),
		workspace.ProvideAsyncProjectorWorkspaceIDIdx(),
	)
	cfg.AddSyncProjectors(collection.ProvideSyncProjectorFactories(appDefBuilder)...)
	uniques.Provide(cfg, appDefBuilder)
	describe.Provide(cfg, vvmAPI.IAppStructsProvider, appDefBuilder)
	signupin.ProvideCmdEnrichPrincipalToken(cfg, appDefBuilder, vvmAPI.IAppTokensFactory)
	cfg.AddCUDValidators(builtin.ProvideRefIntegrityValidator())
}
