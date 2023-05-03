/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys

import (
	"time"

	// "github.com/untillpro/airs-bp3/utils"
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

func Provide(timeFunc func() time.Time, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, hvmAPI vvm.VVMAPI, smtpCfg smtp.Cfg, sep vvm.IStandardExtensionPoints) {
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
	workspace.Provide(cfg, appDefBuilder, hvmAPI.IAppStructsProvider, timeFunc)
	sqlquery.Provide(cfg, appDefBuilder, hvmAPI.IAppStructsProvider, vvm.NumCommandProcessors)
	projectors.ProvideOffsetsDef(appDefBuilder)
	commandprocessor.ProvideJSONFuncParamsDef(appDefBuilder)
	verifier.Provide(cfg, appDefBuilder, hvmAPI.ITokens, hvmAPI.FederationURL, hvmAPI.IAppStructsProvider)
	signupin.ProvideQryRefreshPrincipalToken(cfg, appDefBuilder, hvmAPI.ITokens)
	signupin.ProvideCDocLogin(appDefBuilder)
	invite.Provide(cfg, appDefBuilder, timeFunc)
	cfg.AddAsyncProjectors(
		journal.ProvideWLogDatesAsyncProjectorFactory(),
		workspace.ProvideAsyncProjectorFactoryInvokeCreateWorkspace(hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens),
		workspace.ProvideAsyncProjectorFactoryInvokeCreateWorkspaceID(hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens),
		workspace.ProvideAsyncProjectorInitializeWorkspace(hvmAPI.FederationURL, timeFunc, cfg.Name, sep.EPWSTemplates(), hvmAPI.ITokens),
		verifier.ProvideAsyncProjectorFactory_SendEmailVerificationCode(hvmAPI.FederationURL, smtpCfg),
		invite.ProvideAsyncProjectorApplyInvitationFactory(timeFunc, hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens, smtpCfg),
		invite.ProvideAsyncProjectorApplyJoinWorkspaceFactory(timeFunc, hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens),
		invite.ProvideAsyncProjectorApplyUpdateInviteRolesFactory(timeFunc, hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens, smtpCfg),
		invite.ProvideAsyncProjectorApplyCancelAcceptedInviteFactory(timeFunc, hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens),
		invite.ProvideAsyncProjectorApplyLeaveWorkspaceFactory(timeFunc, hvmAPI.FederationURL, cfg.Name, hvmAPI.ITokens),
	)
	cfg.AddSyncProjectors(
		workspace.ProvideSyncProjectorChildWorkspaceIdxFactory(),
		invite.ProvideSyncProjectorInviteIndexFactory(),
		invite.ProvideSyncProjectorJoinedWorkspaceIndexFactory(),
	)
	cfg.AddSyncProjectors(collection.ProvideSyncProjectorFactories(appDefBuilder)...)
	uniques.Provide(cfg, appDefBuilder)
	describe.Provide(cfg, hvmAPI.IAppStructsProvider, appDefBuilder)
	signupin.ProvideCmdEnrichPrincipalToken(cfg, appDefBuilder, hvmAPI.IAppTokensFactory)
	cfg.AddCUDValidators(builtin.ProvideRefIntegrityValidator())
}
