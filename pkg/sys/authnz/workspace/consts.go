/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"embed"

	"github.com/voedger/voedger/pkg/appdef"
)

const (
	field_dummy            = "dummy"
	field_TemplateName     = "TemplateName"
	Field_OwnerWSID        = "OwnerWSID"
	Field_OwnerQName       = "OwnerQName"
	Field_OwnerID          = "OwnerID"
	Field_OwnerApp         = "OwnerApp"
	Field_TemplateParams   = "TemplateParams"
	Field_CreateError      = "CreateError"
	Field_InitStartedAtMs  = "InitStartedAtMs"
	Field_ChildWorkspaceID = "ChildWorkspaceID"
	workspace              = "Workspace"
)

var (
	QNameViewChildWorkspaceIdx     = appdef.NewQName(appdef.SysPackage, "ChildWorkspaceIdx")
	QNameViewWorkspaceIDIdx        = appdef.NewQName(appdef.SysPackage, "WorkspaceIDIdx")
	QNameQueryChildWorkspaceByName = appdef.NewQName(appdef.SysPackage, "QueryChildWorkspaceByName")
	QNameCDocWorkspaceID           = appdef.NewQName(appdef.SysPackage, "WorkspaceID")
	qNameAPInitializeWorkspace     = appdef.NewQName(appdef.SysPackage, "InitializeWorkspace")
	qNameAPInvokeCreateWorkspaceID = appdef.NewQName(appdef.SysPackage, "InvokeCreateWorkspaceID")
	qNameAPInvokeCreateWorkspace   = appdef.NewQName(appdef.SysPackage, "InvokeCreateWorkspace")
)

//go:embed postinit/*
var Postinit embed.FS

const (
	pathCap    = 3
	ticketsCap = 3
	base       = 10
)
