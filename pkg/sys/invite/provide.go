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

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder, timeFunc coreutils.TimeFunc,
	federation federation.IFederation, itokens itokens.ITokens, smtpCfg smtp.Cfg) {
	provideCmdInitiateInvitationByEMail(sprb, timeFunc)
	provideCmdInitiateJoinWorkspace(sprb, timeFunc)
	provideCmdInitiateUpdateInviteRoles(sprb, timeFunc)
	provideCmdInitiateCancelAcceptedInvite(sprb, timeFunc)
	provideCmdInitiateLeaveWorkspace(sprb, timeFunc)
	provideCmdCancelSentInvite(sprb, timeFunc)
	provideCmdCreateJoinedWorkspace(sprb)
	provideCmdUpdateJoinedWorkspaceRoles(sprb)
	provideCmdDeactivateJoinedWorkspace(sprb)
	sprb.AddAsyncProjectors(
		asyncProjectorApplyInvitation(timeFunc, federation, itokens, smtpCfg),
		asyncProjectorApplyJoinWorkspace(timeFunc, federation, itokens),
		asyncProjectorApplyUpdateInviteRoles(timeFunc, federation, itokens, smtpCfg),
		asyncProjectorApplyCancelAcceptedInvite(timeFunc, federation, itokens),
		asyncProjectorApplyLeaveWorkspace(timeFunc, federation, itokens),
	)
	sprb.AddSyncProjectors(
		syncProjectorInviteIndex(),
		syncProjectorJoinedWorkspaceIndex(),
		applyViewSubjectsIdx(),
	)
}
