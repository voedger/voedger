/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package sysshared

const (
// workspace package
// Field_InitError         = "InitError"
// Field_InitCompletedAtMs = "InitCompletedAtMs"

// invite package
// Field_WSName      = "WSName"
// Field_SubjectKind = "SubjectKind"
// Field_ProfileWSID = "ProfileWSID"

// workspace package (for deactivate workspace)
// Field_Status = "Status"
)

var (
// workspace package
// QNameCDocWorkspaceDescriptor  = appdef.NewQName(appdef.SysPackage, "WorkspaceDescriptor")
// QNameCommandCreateWorkspaceID = appdef.NewQName(appdef.SysPackage, "CreateWorkspaceID")
// QNameCommandCreateWorkspace   = appdef.NewQName(appdef.SysPackage, "CreateWorkspace")
// QNameCommandInit              = appdef.NewQName(appdef.SysPackage, "Init")
// QNameCommandImport            = appdef.NewQName(appdef.SysPackage, "Import")

// blobber package
// QNameCommandUploadBLOBHelper = appdef.NewQName(appdef.SysPackage, "UploadBLOBHelper")
// QNameWDocBLOB                = appdef.NewQName(appdef.SysPackage, "BLOB")

// invite package
// QNameCDocJoinedWorkspace = appdef.NewQName(appdef.SysPackage, "JoinedWorkspace")
// QNameCDocSubject         = appdef.NewQName(appdef.SysPackage, "Subject")
// в sys нельзя, т.к. sys->signupin->vvm(там FederationURLType нужен)->cmd->(нужны константы)sys
)

// const (
// 	WorkspaceStatus_Active WorkspaceStatus = iota
// 	WorkspaceStatus_ToBeDeactivated
// 	WorkspaceStatus_Inactive
// )
