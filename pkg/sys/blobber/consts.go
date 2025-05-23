/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import (
	"github.com/voedger/voedger/pkg/appdef"
)

const (
	fldStatus           = "status"
	fldOwnerRecord      = "OwnerRecord"
	fldOwnerRecordField = "OwnerRecordField"
)

var (
	QNameCommandUploadBLOBHelper = appdef.NewQName(appdef.SysPackage, "UploadBLOBHelper")
	QNameWDocBLOB                = appdef.NewQName(appdef.SysPackage, "BLOB")
)
