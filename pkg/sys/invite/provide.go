/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, timeFunc coreutils.TimeFunc,
	federation coreutils.IFederation, itokens itokens.ITokens, smtpCfg smtp.Cfg) {
	provideCmdInitiateInvitationByEMail(cfg, appDefBuilder, timeFunc)
	provideCmdInitiateJoinWorkspace(cfg, appDefBuilder, timeFunc)
	provideCmdInitiateUpdateInviteRoles(cfg, appDefBuilder, timeFunc)
	provideCmdInitiateCancelAcceptedInvite(cfg, appDefBuilder, timeFunc)
	provideCmdInitiateLeaveWorkspace(cfg, timeFunc)
	provideCmdCancelSentInvite(cfg, appDefBuilder, timeFunc)
	provideCmdCreateJoinedWorkspace(cfg, appDefBuilder)
	provideCmdUpdateJoinedWorkspaceRoles(cfg, appDefBuilder)
	provideCmdDeactivateJoinedWorkspace(cfg, appDefBuilder)
	provideCDocSubject(cfg, appDefBuilder)
	provideViewInviteIndex(appDefBuilder)
	provideViewJoinedWorkspaceIndex(appDefBuilder)
	appDefBuilder.AddObject(qNameAPApplyCancelAcceptedInvite)
	appDefBuilder.AddObject(qNameAPApplyInvitation)
	appDefBuilder.AddObject(qNameAPApplyJoinWorkspace)
	appDefBuilder.AddObject(qNameAPApplyLeaveWorkspace)
	appDefBuilder.AddObject(qNameAPApplyUpdateInviteRoles)
	cfg.AddAsyncProjectors(
		provideAsyncProjectorApplyInvitationFactory(timeFunc, federation, cfg.Name, itokens, smtpCfg),
		provideAsyncProjectorApplyJoinWorkspaceFactory(timeFunc, federation, cfg.Name, itokens),
		provideAsyncProjectorApplyUpdateInviteRolesFactory(timeFunc, federation, cfg.Name, itokens, smtpCfg),
		provideAsyncProjectorApplyCancelAcceptedInviteFactory(timeFunc, federation, cfg.Name, itokens),
		provideAsyncProjectorApplyLeaveWorkspaceFactory(timeFunc, federation, cfg.Name, itokens),
	)
	cfg.AddSyncProjectors(
		provideSyncProjectorInviteIndexFactory(),
		provideSyncProjectorJoinedWorkspaceIndexFactory(),
	)
}
