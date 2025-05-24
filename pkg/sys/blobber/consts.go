/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import (
	"github.com/voedger/voedger/pkg/appdef"
)

const (
	field_status           = "status"
	field_OwnerRecord      = "OwnerRecord"
	field_OwnerRecordField = "OwnerRecordField"
	field_OwnerRecordID    = "OwnerRecordID"
)

var (
	QNameCommandUploadBLOBHelper = appdef.NewQName(appdef.SysPackage, "UploadBLOBHelper")
	QNameWDocBLOB                = appdef.NewQName(appdef.SysPackage, "BLOB")
)
