/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func Provide(cfg *istructsmem.AppConfigType, timeFunc coreutils.TimeFunc,
	federation federation.IFederation, itokens itokens.ITokens, smtpCfg smtp.Cfg) {
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
		asyncProjectorApplyInvitation(timeFunc, federation, itokens, smtpCfg),
		asyncProjectorApplyJoinWorkspace(timeFunc, federation, itokens),
		asyncProjectorApplyUpdateInviteRoles(timeFunc, federation, itokens, smtpCfg),
		asyncProjectorApplyCancelAcceptedInvite(timeFunc, federation, itokens),
		asyncProjectorApplyLeaveWorkspace(timeFunc, federation, itokens),
	)
	cfg.AddSyncProjectors(
		syncProjectorInviteIndex(),
		syncProjectorJoinedWorkspaceIndex(),
		applyViewSubjectsIdx(),
	)
}
