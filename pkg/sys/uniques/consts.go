/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import "github.com/voedger/voedger/pkg/appdef"

const (
	field_ID         = "ID"
	field_ValuesHash = "ValuesHash"
	field_QName      = "QName"
	field_Values     = "Values"
	zeroByte         = byte(0)
)

var (
	qNameApplyUniques = appdef.NewQName(appdef.SysPackage, "ApplyUniques")
	qNameViewUniques  = appdef.NewQName(appdef.SysPackage, "Uniques")

	// FIXME: see https://github.com/voedger/voedger/issues/2208
	qnameRecordStorage = appdef.NewQName(appdef.SysPackage, "Record")
	qnameViewStorage   = appdef.NewQName(appdef.SysPackage, "View")
	field_RecordID     = "ID"
	// End FIXME

)
