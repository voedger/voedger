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
)
