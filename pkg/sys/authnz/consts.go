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
	// Field_InitError                = "InitError"
	// Field_InitCompletedAtMs        = "InitCompletedAtMs"
	Field_СreatedAtMs = "CreatedAtMs"
	Field_WSName      = "WSName"
	Field_WSKind      = "WSKind"
)

var (
	QNameCDoc_WorkspaceKind_UserProfile            = appdef.NewQName(appdef.SysPackage, "UserProfile")
	QNameCDoc_WorkspaceKind_DeviceProfile          = appdef.NewQName(appdef.SysPackage, "DeviceProfile")
	QNameCDoc_WorkspaceKind_AppWorkspace           = appdef.NewQName(appdef.SysPackage, "AppWorkspace")
	QNameCDocLogin                                 = appdef.NewQName(appdef.SysPackage, "Login")
	QNameCDocChildWorkspace                        = appdef.NewQName(appdef.SysPackage, "ChildWorkspace")
	QNameCommandInitChildWorkspace                 = appdef.NewQName(appdef.SysPackage, "InitChildWorkspace")
	QNameCommandCreateLogin                        = appdef.NewQName(appdef.SysPackage, "CreateLogin")
	QNameCommandResetPasswordByEmail               = appdef.NewQName(appdef.SysPackage, "ResetPasswordByEmail")
	QNameCommandResetPasswordByEmailUnloggedParams = appdef.NewQName(appdef.SysPackage, "ResetPasswordByEmailUnloggedParams")
	// QNameCDocWorkspaceDescriptor                      = appdef.NewQName(appdef.SysPackage, "WorkspaceDescriptor")
	// QNameCommandCreateWorkspace                       = appdef.NewQName(appdef.SysPackage, "CreateWorkspace")
	// QNameCommandCreateWorkspaceID                     = appdef.NewQName(appdef.SysPackage, "CreateWorkspaceID")
	QNameQueryInitiateResetPasswordByEmail            = appdef.NewQName(appdef.SysPackage, "InitiateResetPasswordByEmail")
	QNameQueryIssueVerifiedValueTokenForResetPassword = appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueTokenForResetPassword")
)
