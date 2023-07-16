/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import "github.com/voedger/voedger/pkg/appdef"

const (
	nameCDOC      = "CDoc"
	nameODOC      = "ODoc"
	nameWDOC      = "WDoc"
	nameSingleton = "Singleton"
	nameCRecord   = "CRecord"
	nameORecord   = "ORecord"
	nameWRecord   = "WRecord"
)

const (
	sysInt     = "int"
	sysInt32   = "int32"
	sysInt64   = "int64"
	sysFloat   = "float"
	sysFloat32 = "float32"
	sysFloat64 = "float64"
	sysQName   = "qname"
	sysBool    = "bool"
	sysString  = "text"
	sysBytes   = "bytes"
	sysBlob    = "blob"

	sysVoid = "void"
)

const maxNestedTableContainerOccurrences = 100 // FIXME: 100 container occurrences

var canNotReferenceTo = map[appdef.DefKind][]appdef.DefKind{
	appdef.DefKind_ODoc:    {},
	appdef.DefKind_ORecord: {},
	appdef.DefKind_WDoc:    {appdef.DefKind_ODoc, appdef.DefKind_ORecord},
	appdef.DefKind_WRecord: {appdef.DefKind_ODoc, appdef.DefKind_ORecord},
	appdef.DefKind_CDoc:    {appdef.DefKind_WDoc, appdef.DefKind_WRecord, appdef.DefKind_ODoc, appdef.DefKind_ORecord},
	appdef.DefKind_CRecord: {appdef.DefKind_WDoc, appdef.DefKind_WRecord, appdef.DefKind_ODoc, appdef.DefKind_ORecord},
}
