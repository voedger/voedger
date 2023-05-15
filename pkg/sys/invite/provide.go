/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, timeFunc func() time.Time) {
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
	provideCDocInvite(cfg, appDefBuilder)
	provideCDocJoinedWorkspace(appDefBuilder)
	provideViewInviteIndex(appDefBuilder)
	provideViewJoinedWorkspaceIndex(appDefBuilder)
	appDefBuilder.AddStruct(qNameAPApplyCancelAcceptedInvite, appdef.DefKind_Object)
	appDefBuilder.AddStruct(qNameAPApplyInvitation, appdef.DefKind_Object)
	appDefBuilder.AddStruct(qNameAPApplyJoinWorkspace, appdef.DefKind_Object)
	appDefBuilder.AddStruct(qNameAPApplyLeaveWorkspace, appdef.DefKind_Object)
	appDefBuilder.AddStruct(qNameAPApplyUpdateInviteRoles, appdef.DefKind_Object)
}
