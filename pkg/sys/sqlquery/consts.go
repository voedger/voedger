/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	base          = 10
	bitSize64     = 64
	DefaultLimit  = 100
	DefaultOffset = istructs.FirstOffset
	field_Query   = "Query"
	flag_WSID     = "--wsid="
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
