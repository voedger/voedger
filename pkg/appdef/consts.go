/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "math"

// Maximum identifier length
const MaxIdentLen = 255

const (
	// System package name
	SysPackage = "sys"

	// Used as delimiter in qualified names
	QNameQualifierChar = "."

	// Used as prefix for names of system fields
	SystemPackagePrefix = SysPackage + QNameQualifierChar

	// System package path
	SysPackagePath = "voedger.com/packages/sys"
)

// System QName
const SysWorkspaceName = "Workspace"

var SysWorkspaceQName = NewQName(SysPackage, SysWorkspaceName)

const AnyName = "ANY"

// QNameANY is substitution denotes that a Function param or result can be any type
//
// See #858 (Support QNameAny as function result)
var QNameANY = NewQName(SysPackage, AnyName)

// QNameRoleSystem is role for system.
var QNameRoleSystem = NewQName(SysPackage, "System")

var QNameRoleDeveloper = NewQName(SysPackage, "Developer")

const (
	// System application owner name
	SysOwner = "sys"

	// Char to separate application owner (provider) from application name
	AppQNameQualifierChar = "/"
)

// Maximum fields per one structured type
const MaxTypeFieldCount = 65536

// System field names
const (
	SystemField_ID        = SystemPackagePrefix + "ID"
	SystemField_ParentID  = SystemPackagePrefix + "ParentID"
	SystemField_IsActive  = SystemPackagePrefix + "IsActive"
	SystemField_Container = SystemPackagePrefix + "Container"
	SystemField_QName     = SystemPackagePrefix + "QName"
)

// System data type names
var (
	SysData_int8     QName = SysDataName(DataKind_int8)  // #3434 [~server.vsql.smallints/cmp.AppDef~impl]
	SysData_int16    QName = SysDataName(DataKind_int16) // #3434 [~server.vsql.smallints/cmp.AppDef~impl]
	SysData_int32    QName = SysDataName(DataKind_int32)
	SysData_int64    QName = SysDataName(DataKind_int64)
	SysData_float32  QName = SysDataName(DataKind_float32)
	SysData_float64  QName = SysDataName(DataKind_float64)
	SysData_bytes    QName = SysDataName(DataKind_bytes)
	SysData_String   QName = SysDataName(DataKind_string)
	SysData_QName    QName = SysDataName(DataKind_QName)
	SysData_bool     QName = SysDataName(DataKind_bool)
	SysData_RecordID QName = SysDataName(DataKind_RecordID)
)

// Maximum containers per one structured type
const MaxTypeContainerCount = 65536

// Maximum fields per one unique
const MaxTypeUniqueFieldsCount = 256

// Maximum uniques
const MaxTypeUniqueCount = 100

// Maximum string and bytes data length
const MaxFieldLength = uint16(math.MaxUint16)

// Default string and bytes data max length.
//
// This value is used for MaxLen() constraint in system data types `sys.string` and `sys.bytes`.
const DefaultFieldMaxLength = uint16(255)
