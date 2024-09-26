/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"github.com/voedger/voedger/pkg/appdef"
)

const (
	DefaultLimit = 100
	field_Query  = "Query"
)

var (
	plog    = appdef.NewQName(appdef.SysPackage, "plog")
	plogDef = map[string]bool{
		"PlogOffset":     true,
		"QName":          true,
		"ArgumentObject": true,
		"CUDs":           true,
		"RegisteredAt":   true,
		"Synced":         true,
		"DeviceID":       true,
		"SyncedAt":       true,
		"Error":          true,
		"Workspace":      true,
		"WLogOffset":     true,
	}
	wlog    = appdef.NewQName(appdef.SysPackage, "wlog")
	wlogDef = map[string]bool{
		"WlogOffset":     true,
		"QName":          true,
		"ArgumentObject": true,
		"CUDs":           true,
		"RegisteredAt":   true,
		"Synced":         true,
		"DeviceID":       true,
		"SyncedAt":       true,
		"Error":          true,
	}
)
