/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"net/http"

	"github.com/untillpro/voedger/pkg/istructs"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

const (
	Field_JSONSchemaBody = "Body"
	intentsLimit         = 128
)

var (
	ViewQNamePLogKnownOffsets = istructs.NewQName(istructs.SysPackage, "PLogKnownOffsets")
	ViewQNameWLogKnownOffsets = istructs.NewQName(istructs.SysPackage, "WLogKnownOffsets")
	errWSNotInited            = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace is not initialized")
)

// TODO: should be in separate package
const (
	Field_InitError         = "InitError"
	Field_InitCompletedAtMs = "InitCompletedAtMs"
)

var (
	QNameCDocWorkspaceDescriptor  = istructs.NewQName(istructs.SysPackage, "WorkspaceDescriptor")
	QNameCommandCreateWorkspaceID = istructs.NewQName(istructs.SysPackage, "CreateWorkspaceID")
	QNameCommandCreateWorkspace   = istructs.NewQName(istructs.SysPackage, "CreateWorkspace")
	QNameCommandUploadBLOBHelper  = istructs.NewQName(istructs.SysPackage, "UploadBLOBHelper")
	QNameWDocBLOB                 = istructs.NewQName(istructs.SysPackage, "BLOB")
	QNameCommandInit              = istructs.NewQName(istructs.SysPackage, "Init")
	QNameCommandImport            = istructs.NewQName(istructs.SysPackage, "Import")
)
