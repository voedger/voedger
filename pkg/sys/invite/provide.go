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
	"github.com/voedger/voedger/pkg/utils/federation"
)

func Provide(sr istructsmem.IStatelessResources, timeFunc coreutils.TimeFunc,
	federation federation.IFederation, itokens itokens.ITokens, smtpCfg smtp.Cfg) {
	provideCmdInitiateInvitationByEMail(sr, timeFunc)
	provideCmdInitiateJoinWorkspace(sr, timeFunc)
	provideCmdInitiateUpdateInviteRoles(sr, timeFunc)
	provideCmdInitiateCancelAcceptedInvite(sr, timeFunc)
	provideCmdInitiateLeaveWorkspace(sr, timeFunc)
	provideCmdCancelSentInvite(sr, timeFunc)
	provideCmdCreateJoinedWorkspace(sr)
	provideCmdUpdateJoinedWorkspaceRoles(sr)
	provideCmdDeactivateJoinedWorkspace(sr)
	sr.AddProjectors(appdef.SysPackagePath,
		asyncProjectorApplyInvitation(timeFunc, federation, itokens, smtpCfg),
		asyncProjectorApplyJoinWorkspace(timeFunc, federation, itokens),
		asyncProjectorApplyUpdateInviteRoles(timeFunc, federation, itokens, smtpCfg),
		asyncProjectorApplyCancelAcceptedInvite(timeFunc, federation, itokens),
		asyncProjectorApplyLeaveWorkspace(timeFunc, federation, itokens),
		syncProjectorInviteIndex(),
		syncProjectorJoinedWorkspaceIndex(),
		applyViewSubjectsIdx(),
	)
}
