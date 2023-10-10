/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"embed"

	"github.com/voedger/voedger/pkg/appdef"
)

var (
	// Deprecated: use c.sys.CUD instead. Kept to not to break existing events only
	QNameCommandInit = appdef.NewQName(appdef.SysPackage, "Init")
	//go:embed schema.sql
	schemaBuiltinFS          embed.FS
	QNameViewRecordsRegistry = appdef.NewQName(appdef.SysPackage, "RecordsRegistry")
)

const (
	field_ExistingQName = "ExistingQName"
	field_NewQName      = "NewQName"
	MaxCUDs             = 351 // max rawID in perftest template is 351
	field_IDHi          = "IDHi"
	field_ID            = "ID"
	field_WLogOffset    = "WLogOffset"
	field_QName         = "QName"
)
