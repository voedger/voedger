/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import "github.com/voedger/voedger/pkg/appdef"

const (
	filterKind_Eq    = "eq"
	filterKind_NotEq = "notEq"
	filterKind_Gt    = "gt"
	filterKind_Lt    = "lt"
	filterKind_And   = "and"
	filterKind_Or    = "or"
)

const (
	minNormalFloat64 = 0x1.0p-1022
	rootDocument     = ""
)

var (
	// test only constants
	qNamePosDepartment       = appdef.NewQName("pos", "Department")
	qNamePosDepartmentResult = appdef.NewQName("pos", "DepartmentResult")
	qNameXLowerCase          = appdef.NewQName("x", "lower_case")
)
