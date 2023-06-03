/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

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
	sysBytes   = "blob"
)

const maxNestedTableContainerOccurrences = 100 // FIXME: 100 container occurrences
