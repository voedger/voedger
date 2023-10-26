/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package authnz

import (
	"github.com/voedger/voedger/pkg/appdef"
)

const (
	Field_DisplayName              = "DisplayName"
	Field_WSID                     = "WSID"
	Field_PrincipalToken           = "PrincipalToken"
	Field_WSError                  = "WSError"
	Field_SubjectKind              = "SubjectKind"
	Field_WSKindInitializationData = "WSKindInitializationData"
	Field_WSClusterID              = "WSClusterID"
	Field_ProfileClusterID         = "ProfileCluster"
	Field_LoginHash                = "LoginHash"
	Field_Login                    = "Login"
	Field_СreatedAtMs              = "CreatedAtMs"
	Field_WSName                   = "WSName"
	Field_WSKind                   = "WSKind"
)

var (
	QNameCDoc_WorkspaceKind_UserProfile               = appdef.NewQName(appdef.SysPackage, "UserProfile")
	QNameCDoc_WorkspaceKind_DeviceProfile             = appdef.NewQName(appdef.SysPackage, "DeviceProfile")
	QNameCDoc_WorkspaceKind_AppWorkspace              = appdef.NewQName(appdef.SysPackage, "AppWorkspace")
	QNameCDocLogin                                    = appdef.NewQName(appdef.SysPackage, "Login")
	QNameCDocChildWorkspace                           = appdef.NewQName(appdef.SysPackage, "ChildWorkspace")
	QNameCommandInitChildWorkspace                    = appdef.NewQName(appdef.SysPackage, "InitChildWorkspace")
	QNameCommandCreateLogin                           = appdef.NewQName(appdef.SysPackage, "CreateLogin")
	QNameCommandResetPasswordByEmail                  = appdef.NewQName(appdef.SysPackage, "ResetPasswordByEmail")
	QNameCommandResetPasswordByEmailUnloggedParams    = appdef.NewQName(appdef.SysPackage, "ResetPasswordByEmailUnloggedParams")
	QNameQueryInitiateResetPasswordByEmail            = appdef.NewQName(appdef.SysPackage, "InitiateResetPasswordByEmail")
	QNameQueryIssueVerifiedValueTokenForResetPassword = appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueTokenForResetPassword")

	// at workspace is wrong: deactivate workspace uses invite.QNameCDocSubject, invite uses cdoc.sys.WorkspaceDescriptor -> import cycle
	QNameCDocWorkspaceDescriptor = appdef.NewQName(appdef.SysPackage, "WorkspaceDescriptor")

	// should be here because: collection->qp(tests)->workspace(checkISWSActive)->collection(read out subjects) -> import cycle
	//                               breaking this ^^^
	Field_Status = "Status"
)

const (
	// should be here because: collection->qp(tests)->workspace(checkISWSActive)->collection(read out subjects) -> import cycle
	//                               breaking this ^^^
	WorkspaceStatus_Active WorkspaceStatus = iota
	WorkspaceStatus_ToBeDeactivated
	WorkspaceStatus_Inactive
)
