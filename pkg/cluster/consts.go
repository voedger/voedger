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
	field_NewID            = "NewID"
)

var (
	qNameWDocApp          = appdef.NewQName(ClusterPackage, "App")
	plog                  = appdef.NewQName(appdef.SysPackage, "PLog")
	wlog                  = appdef.NewQName(appdef.SysPackage, "WLog")
	qNameVSqlUpdateResult = appdef.NewQName(ClusterPackage, "VSqlUpdateResult")
	updateDeniedFields    = map[string]bool{
		appdef.SystemField_ID:    true,
		appdef.SystemField_QName: true,
	}
	allowedOpKinds = map[dml.OpKind]bool{
		dml.OpKind_UnloggedInsert:  true,
		dml.OpKind_UnloggedUpdate:  true,
		dml.OpKind_UpdateCorrupted: true,
		dml.OpKind_UpdateTable:     true,
		dml.OpKind_InsertTable:     true,
	}

	// if the name is like a sql identifier e.g. `Int` then the parser makes it lowered
	sqlFieldNamesUnlowered = map[string]string{
		"int":   "Int",
		"bool":  "Bool",
		"bytes": "Bytes",
	}

	allowedDocsTypeKinds = map[appdef.TypeKind]bool{
		appdef.TypeKind_CDoc: true,
		appdef.TypeKind_WDoc: true,
	}
)
