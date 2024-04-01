/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "math"

// Maximum identifier length
const MaxIdentLen = 255

// Any name
const AnyName = "ANY"

const (
	// System package name
	SysPackage = "sys"

	// Used as delimiter in qualified names
	QNameQualifierChar = "."

	// Used as prefix for names of system fields and containers
	SystemPackagePrefix = SysPackage + QNameQualifierChar

	// System package path
	SysPackagePath = "voedger.com/packages/sys"
)

// QNameANY denotes that a Function param or result can be any type
//
// See #858 (Support QNameAny as function result)
var QNameANY = NewQName(SysPackage, AnyName)

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
