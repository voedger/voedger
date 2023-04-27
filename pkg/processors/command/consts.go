/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	Field_JSONSchemaBody = "Body"
	intentsLimit         = 128
)

var (
	ViewQNamePLogKnownOffsets = appdef.NewQName(appdef.SysPackage, "PLogKnownOffsets")
	ViewQNameWLogKnownOffsets = appdef.NewQName(appdef.SysPackage, "WLogKnownOffsets")
	errWSNotInited            = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace is not initialized")
)

// TODO: should be in a separate package
const (
	Field_InitError         = "InitError"
	Field_InitCompletedAtMs = "InitCompletedAtMs"
)

// TODO: should be in a separate package
var (
	QNameCDocWorkspaceDescriptor  = appdef.NewQName(appdef.SysPackage, "WorkspaceDescriptor")
	QNameCommandCreateWorkspaceID = appdef.NewQName(appdef.SysPackage, "CreateWorkspaceID")
	QNameCommandCreateWorkspace   = appdef.NewQName(appdef.SysPackage, "CreateWorkspace")
	QNameCommandUploadBLOBHelper  = appdef.NewQName(appdef.SysPackage, "UploadBLOBHelper")
	QNameWDocBLOB                 = appdef.NewQName(appdef.SysPackage, "BLOB")
	QNameCommandInit              = appdef.NewQName(appdef.SysPackage, "Init")
	QNameCommandImport            = appdef.NewQName(appdef.SysPackage, "Import")
)
