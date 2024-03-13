/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, timeFunc coreutils.TimeFunc,
	federation coreutils.IFederation, itokens itokens.ITokens, smtpCfg smtp.Cfg) {
	provideCmdInitiateInvitationByEMail(cfg, timeFunc)
	provideCmdInitiateJoinWorkspace(cfg, timeFunc)
	provideCmdInitiateUpdateInviteRoles(cfg, timeFunc)
	provideCmdInitiateCancelAcceptedInvite(cfg, timeFunc)
	provideCmdInitiateLeaveWorkspace(cfg, timeFunc)
	provideCmdCancelSentInvite(cfg, timeFunc)
	provideCmdCreateJoinedWorkspace(cfg)
	provideCmdUpdateJoinedWorkspaceRoles(cfg)
	provideCmdDeactivateJoinedWorkspace(cfg)
	cfg.AddAsyncProjectors(
		asyncProjectorApplyInvitation(timeFunc, federation, cfg.Name, itokens, smtpCfg),
		asyncProjectorApplyJoinWorkspace(timeFunc, federation, cfg.Name, itokens),
		asyncProjectorApplyUpdateInviteRoles(timeFunc, federation, cfg.Name, itokens, smtpCfg),
		asyncProjectorApplyCancelAcceptedInvite(timeFunc, federation, cfg.Name, itokens),
		asyncProjectorApplyLeaveWorkspace(timeFunc, federation, cfg.Name, itokens),
	)
	cfg.AddSyncProjectors(
		syncProjectorInviteIndex(),
		syncProjectorJoinedWorkspaceIndex(),
		applyViewSubjectsIdx(),
	)
}
