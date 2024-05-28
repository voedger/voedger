/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"embed"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/dml"
)

//go:embed appws.vsql
var schemaFS embed.FS

const (
	ClusterPackage         = "cluster"
	ClusterPackageFQN      = "github.com/voedger/voedger/pkg/" + ClusterPackage
	Field_ClusterAppID     = "ClusterAppID"
	Field_AppQName         = "AppQName"
	Field_NumPartitions    = "NumPartitions"
	Field_NumAppWorkspaces = "NumAppWorkspaces"
	field_Query            = "Query"
	bitSize64              = 64
	base10                 = 10
)

var (
	qNameWDocApp       = appdef.NewQName(ClusterPackage, "App")
	plog               = appdef.NewQName(appdef.SysPackage, "PLog")
	wlog               = appdef.NewQName(appdef.SysPackage, "WLog")
	updateDeniedFields = map[string]bool{
		appdef.SystemField_ID:    true,
		appdef.SystemField_QName: true,
	}
	allowedDMLKinds = map[dml.OpKind]bool{
		dml.OpKind_DirectInsert:    true,
		dml.OpKind_DirectUpdate:    true,
		dml.OpKind_UpdateCorrupted: true,
		dml.OpKind_UpdateTable:     true,
	}

	// if the name is like a sql identifier e.g. `Int` then the parser makes it lowered
	sqlFieldNamesUnlowered = map[string]string{
		"int":   "Int",
		"bool":  "Bool",
		"bytes": "Bytes",
	}
)
