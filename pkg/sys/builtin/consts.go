/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
)

var (
	// Deprecated: use c.sys.CUD instead. Kept to not to break existing events only
	QNameCommandInit = appdef.NewQName(appdef.SysPackage, "Init")
)

const (
	field_ExistingQName = "ExistingQName"
	field_NewQName      = "NewQName"
	MaxCUDs             = 100
)

// Records registry view
var (
	QNameViewRecordsRegistry      = sys.RecordsRegistryView.Name
	qNameRecordsRegistryProjector = appdef.NewQName(appdef.SysPackage, "RecordsRegistryProjector")
	Field_IDHi                    = sys.RecordsRegistryView.Fields.IDHi
	Field_ID                      = sys.RecordsRegistryView.Fields.ID
	Field_WLogOffset              = sys.RecordsRegistryView.Fields.WLogOffset
	field_QName                   = sys.RecordsRegistryView.Fields.QName
	// not yet used: field_IsActive                = sys.RecordsRegistryView.Fields.IsActive
)
