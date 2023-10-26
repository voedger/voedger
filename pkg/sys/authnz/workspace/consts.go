/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
)

const (
	field_dummy                                     = "dummy"
	field_TemplateName                              = "TemplateName"
	Field_OwnerWSID                                 = "OwnerWSID"
	Field_OwnerQName                                = "OwnerQName"
	Field_OwnerID                                   = "OwnerID"
	Field_OwnerApp                                  = "OwnerApp"
	Field_TemplateParams                            = "TemplateParams"
	Field_CreateError                               = "CreateError"
	Field_InitStartedAtMs                           = "InitStartedAtMs"
	Field_ChildWorkspaceID                          = "ChildWorkspaceID"
	workspace                                       = "Workspace"
	fldDummy1                                       = "dummy1"
	fldDummy2                                       = "dummy2"
	fldNextBaseWSID                                 = "NextBaseWSID"
	field_InvitedToWSID                             = "InvitedToWSID"
	field_IDOfCDocWorkspaceID                       = "IDOfCDocWorkspaceID"
	Field_InitError                                 = "InitError"
	Field_InitCompletedAtMs                         = "InitCompletedAtMs"
	EPWSTemplates             extensionpoints.EPKey = "WSTemplates"
)

var (
	QNameViewChildWorkspaceIdx             = appdef.NewQName(appdef.SysPackage, "ChildWorkspaceIdx")
	QNameProjectorChildWorkspaceIdx        = appdef.NewQName(appdef.SysPackage, "ProjectorChildWorkspaceIdx")
	QNameViewWorkspaceIDIdx                = appdef.NewQName(appdef.SysPackage, "WorkspaceIDIdx")
	QNameProjectorViewWorkspaceIDIdx       = appdef.NewQName(appdef.SysPackage, "ProjectorWorkspaceIDIdx")
	QNameQueryChildWorkspaceByName         = appdef.NewQName(appdef.SysPackage, "QueryChildWorkspaceByName")
	QNameCDocWorkspaceID                   = appdef.NewQName(appdef.SysPackage, "WorkspaceID")
	qNameAPInitializeWorkspace             = appdef.NewQName(appdef.SysPackage, "InitializeWorkspace")
	qNameAPInvokeCreateWorkspaceID         = appdef.NewQName(appdef.SysPackage, "InvokeCreateWorkspaceID")
	qNameAPInvokeCreateWorkspace           = appdef.NewQName(appdef.SysPackage, "InvokeCreateWorkspace")
	ViewQNameNextBaseWSID                  = appdef.NewQName(appdef.SysPackage, "NextBaseWSID")
	qNameCmdInitiateDeactivateWorkspace    = appdef.NewQName(appdef.SysPackage, "InitiateDeactivateWorkspace")
	qNameProjectorApplyDeactivateWorkspace = appdef.NewQName(appdef.SysPackage, "ApplyDeactivateWorkspace")
	QNameCommandCreateWorkspaceID          = appdef.NewQName(appdef.SysPackage, "CreateWorkspaceID")
	QNameCommandCreateWorkspace            = appdef.NewQName(appdef.SysPackage, "CreateWorkspace")
	nextWSIDGlobalLock                     = sync.Mutex{}
)
