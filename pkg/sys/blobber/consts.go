/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import "github.com/voedger/voedger/pkg/appdef"

const (
	FldBLOBID = "blobID"
	fldStatus = "status"
)

var (
	QNameCommandUploadBLOBHelper = appdef.NewQName(appdef.SysPackage, "UploadBLOBHelper")
	QNameWDocBLOB                = appdef.NewQName(appdef.SysPackage, "BLOB")
)
