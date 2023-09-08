/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import "github.com/voedger/voedger/pkg/appdef"

var (
	QNameCommandInit   = appdef.NewQName(appdef.SysPackage, "Init")
	QNameCommandImport = appdef.NewQName(appdef.SysPackage, "Import")
)

const (
	field_ExistingQName = "ExistingQName"
	field_NewQName      = "NewQName"
)
