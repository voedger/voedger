/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/extensionpoints"
)

// sys.NextBaseWSIDView view
var (
	QNameViewNextBaseWSID = sys.NextBaseWSIDView.Name
	fldDummy1             = sys.NextBaseWSIDView.Fields.PartKeyDummy
	fldDummy2             = sys.NextBaseWSIDView.Fields.ClustColDummy
	fldNextBaseWSID       = sys.NextBaseWSIDView.Fields.NextBaseWSID
)

const (
	field_dummy                                     = "dummy"
	field_TemplateName                              = "TemplateName"
	Field_OwnerWSID                                 = "OwnerWSID"
	Field_OwnerID                                   = "OwnerID"
	Field_OwnerApp                                  = "OwnerApp"
	Field_TemplateParams                            = "TemplateParams"
	Field_CreateError                               = "CreateError"
	Field_InitStartedAtMs                           = "InitStartedAtMs"
	Field_ChildWorkspaceID                          = "ChildWorkspaceID"
	workspace                                       = "Workspace"
	field_InvitedToWSID                             = "InvitedToWSID"
	field_IDOfCDocWorkspaceID                       = "IDOfCDocWorkspaceID"
	Field_InitError                                 = "InitError"
	Field_InitCompletedAtMs                         = "InitCompletedAtMs"
	Field_OwnerQName2                               = "OwnerQName2"
	EPWSTemplates             extensionpoints.EPKey = "WSTemplates"

	//Deprecated: use Field_OwnerQName2
	Field_OwnerQName = "OwnerQName"
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
	qNameCmdInitiateDeactivateWorkspace    = appdef.NewQName(appdef.SysPackage, "InitiateDeactivateWorkspace")
	qNameProjectorApplyDeactivateWorkspace = appdef.NewQName(appdef.SysPackage, "ApplyDeactivateWorkspace")
	QNameCommandCreateWorkspaceID          = appdef.NewQName(appdef.SysPackage, "CreateWorkspaceID")
	QNameCommandCreateWorkspace            = appdef.NewQName(appdef.SysPackage, "CreateWorkspace")
	nextWSIDGlobalLock                     = sync.Mutex{}
)
