/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// TODO improvements
// 1. Do all numeric constants start from 1 and set type for it
// 2. Add QName validation in RecordStorage GetBatch method
// 3. Add projector names via appstructs for validation
var (
	qNameCmdInitiateInvitationByEMail    = appdef.NewQName(appdef.SysPackage, "InitiateInvitationByEMail")
	qNameCmdInitiateJoinWorkspace        = appdef.NewQName(appdef.SysPackage, "InitiateJoinWorkspace")
	qNameCmdInitiateUpdateInviteRoles    = appdef.NewQName(appdef.SysPackage, "InitiateUpdateInviteRoles")
	qNameCmdInitiateCancelAcceptedInvite = appdef.NewQName(appdef.SysPackage, "InitiateCancelAcceptedInvite")
	qNameCmdCreateJoinedWorkspace        = appdef.NewQName(appdef.SysPackage, "CreateJoinedWorkspace")
	qNameCmdUpdateJoinedWorkspaceRoles   = appdef.NewQName(appdef.SysPackage, "UpdateJoinedWorkspaceRoles")
	qNameCmdDeactivateJoinedWorkspace    = appdef.NewQName(appdef.SysPackage, "DeactivateJoinedWorkspace")
	qNameCmdInitiateLeaveWorkspace       = appdef.NewQName(appdef.SysPackage, "InitiateLeaveWorkspace")
	qNameCmdCancelSentInvite             = appdef.NewQName(appdef.SysPackage, "CancelSentInvite")
	QNameCDocInvite                      = appdef.NewQName(appdef.SysPackage, "Invite")
	qNameViewInviteIndex                 = appdef.NewQName(appdef.SysPackage, "InviteIndexView")
	qNameProjectorInviteIndex            = appdef.NewQName(appdef.SysPackage, "ProjectorInviteIndex")
	QNameViewJoinedWorkspaceIndex        = appdef.NewQName(appdef.SysPackage, "JoinedWorkspaceIndexView")
	QNameProjectorJoinedWorkspaceIndex   = appdef.NewQName(appdef.SysPackage, "ProjectorJoinedWorkspaceIndex")
	qNameAPApplyCancelAcceptedInvite     = appdef.NewQName(appdef.SysPackage, "ApplyCancelAcceptedInvite")
	qNameAPApplyInvitation               = appdef.NewQName(appdef.SysPackage, "ApplyInvitation")
	qNameAPApplyJoinWorkspace            = appdef.NewQName(appdef.SysPackage, "ApplyJoinWorkspace")
	qNameAPApplyLeaveWorkspace           = appdef.NewQName(appdef.SysPackage, "ApplyLeaveWorkspace")
	qNameAPApplyUpdateInviteRoles        = appdef.NewQName(appdef.SysPackage, "ApplyUpdateInviteRoles")
	QNameCDocJoinedWorkspace             = appdef.NewQName(appdef.SysPackage, "JoinedWorkspace")
	QNameCDocSubject                     = appdef.NewQName(appdef.SysPackage, "Subject")
	QNameViewSubjectsIdx                 = appdef.NewQName(appdef.SysPackage, "ViewSubjectsIdx")
	QNameApplyViewSubjectsIdx            = appdef.NewQName(appdef.SysPackage, "ApplyViewSubjectsIdx")
)

const (
	field_Email                 = "Email"
	Field_Roles                 = "Roles"
	field_ExpireDatetime        = "ExpireDatetime"
	field_InviteID              = "InviteID"
	field_VerificationCode      = "VerificationCode"
	field_EmailTemplate         = "EmailTemplate"
	field_EmailSubject          = "EmailSubject"
	Field_Login                 = "Login"
	Field_InvitingWorkspaceWSID = "InvitingWorkspaceWSID"
	Field_InviteeProfileWSID    = "InviteeProfileWSID"
	Field_State                 = "State"
	field_Created               = "Created"
	Field_Updated               = "Updated"
	field_SubjectID             = "SubjectID"
	field_Dummy                 = "Dummy"
	field_JoinedWorkspaceID     = "JoinedWorkspaceID"
	Field_SubjectKind           = "SubjectKind"
	Field_ProfileWSID           = "ProfileWSID"
	Field_SubjectID             = "SubjectID"
	Field_LoginHash             = "LoginHash"
	field_ActualLogin           = "ActualLogin"
)

//go:generate stringer -type=State -output=stringer_state.go
type State int32

const (
	State_Null State = iota
	State_ToBeInvited
	State_Invited
	State_ToBeJoined
	State_Joined
	State_ToUpdateRoles
	State_ToBeCancelled
	State_Cancelled
	State_ToBeLeft
	State_Left
	State_FakeLast
)

const (
	value_Dummy_One = int32(17)
	value_Dummy_Two = int32(56)
)

const (
	EmailTemplatePlaceholder_VerificationCode = "${VerificationCode}"
	EmailTemplatePlaceholder_InviteID         = "${InviteID}"
	EmailTemplatePlaceholder_WSID             = "${WSID}"
	EmailTemplatePlaceholder_Roles            = "${Roles}"
	EmailTemplatePlaceholder_WSName           = "${WSName}"
	EmailTemplatePlaceholder_Email            = "${Email}"
)

var (
	inviteValidStates = map[appdef.QName]map[State]bool{
		qNameCmdInitiateInvitationByEMail: {
			State_Cancelled: true,
			State_Left:      true,
			State_Invited:   true,
		},
		qNameCmdInitiateJoinWorkspace: {
			State_Invited: true,
		},
		qNameCmdInitiateUpdateInviteRoles: {
			State_Joined: true,
		},
		qNameCmdInitiateCancelAcceptedInvite: {
			State_Joined: true,
		},
		qNameCmdInitiateLeaveWorkspace: {
			State_Joined: true,
		},
		qNameCmdCancelSentInvite: {
			State_Invited: true,
		},
	}
	reInviteAllowedForState = map[State]bool{
		State_Cancelled: true,
		State_Left:      true,

		// https://github.com/voedger/voedger/issues/3698
		State_ToBeInvited: true,
		State_Invited:     true,
	}
)
