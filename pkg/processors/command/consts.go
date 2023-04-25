/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"net/http"

	"github.com/voedger/voedger/pkg/schemas"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	Field_JSONSchemaBody = "Body"
	intentsLimit         = 128
)

var (
	ViewQNamePLogKnownOffsets = schemas.NewQName(schemas.SysPackage, "PLogKnownOffsets")
	ViewQNameWLogKnownOffsets = schemas.NewQName(schemas.SysPackage, "WLogKnownOffsets")
	errWSNotInited            = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace is not initialized")
)

// TODO: should be in a separate package
const (
	Field_InitError         = "InitError"
	Field_InitCompletedAtMs = "InitCompletedAtMs"
)

// TODO: should be in a separate package
var (
	QNameCDocWorkspaceDescriptor  = schemas.NewQName(schemas.SysPackage, "WorkspaceDescriptor")
	QNameCommandCreateWorkspaceID = schemas.NewQName(schemas.SysPackage, "CreateWorkspaceID")
	QNameCommandCreateWorkspace   = schemas.NewQName(schemas.SysPackage, "CreateWorkspace")
	QNameCommandUploadBLOBHelper  = schemas.NewQName(schemas.SysPackage, "UploadBLOBHelper")
	QNameWDocBLOB                 = schemas.NewQName(schemas.SysPackage, "BLOB")
	QNameCommandInit              = schemas.NewQName(schemas.SysPackage, "Init")
	QNameCommandImport            = schemas.NewQName(schemas.SysPackage, "Import")
)
