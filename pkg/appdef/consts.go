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

// Any names
const (
	AnyName          = "ANY"
	AnyStructureName = "AnyStructure"
	AnyRecordName    = "AnyRecord"
	AnyGDocName      = "AnyGDoc"
	AnyCDocName      = "AnyCDoc"
	AnyWDocName      = "AnyWDoc"
	AnySingletonName = "AnySingleton"
	AnyODocName      = "AnyODoc"
	AnyObjectName    = "AnyObject"
	AnyViewName      = "AnyView"
	AnyExtensionName = "AnyExtension"
	AnyFunctionName  = "AnyFunction"
	AnyCommandName   = "AnyCommand"
	AnyQueryName     = "AnyQuery"
)

// QNameANY is substitution denotes that a Function param or result can be any type
//
// See #858 (Support QNameAny as function result)
var QNameANY = NewQName(SysPackage, AnyName)

// QNameAny××× are substitutions, which used to limit with rates. See #868 (VSQL: Rate limits)
var (
	// QNameAnyStructure is a substitution for any structure type (record, object or view record).
	QNameAnyStructure = NewQName(SysPackage, AnyStructureName)

	// QNameAnyStructure is a substitution for any structure type (record, object or view record).
	QNameAnyRecord = NewQName(SysPackage, AnyRecordName)

	// QNameAnyGDoc is a substitution for any GDoc type.
	QNameAnyGDoc = NewQName(SysPackage, AnyGDocName)

	// QNameAnyCDoc is a substitution for any CDoc type.
	QNameAnyCDoc = NewQName(SysPackage, AnyCDocName)

	// QNameAnyWDoc is a substitution for any WDoc type.
	QNameAnyWDoc = NewQName(SysPackage, AnyWDocName)

	// QNameAnySingleton is a substitution for any singleton type.
	QNameAnySingleton = NewQName(SysPackage, AnySingletonName)

	// QNameAnyODoc is a substitution for any ODoc type.
	QNameAnyODoc = NewQName(SysPackage, AnyODocName)

	// QNameAnyObject is a substitution for any Object type.
	QNameAnyObject = NewQName(SysPackage, AnyObjectName)

	// QNameAnyView is a substitution for any view record type.
	QNameAnyView = NewQName(SysPackage, AnyViewName)

	// QNameAnyExtension is a substitution for any extension type (function or projector).
	QNameAnyExtension = NewQName(SysPackage, AnyExtensionName)

	// QNameAnyFunction is a substitution for any function type (command or query).
	QNameAnyFunction = NewQName(SysPackage, AnyFunctionName)

	// QNameAnyCommand is a substitution for any command type.
	QNameAnyCommand = NewQName(SysPackage, AnyCommandName)

	// QNameAnyQuery is a substitution for any query type.
	QNameAnyQuery = NewQName(SysPackage, AnyQueryName)

	anyTypes = map[QName]IType{
		QNameANY:          AnyType,
		QNameAnyStructure: AnyStructureType,
		QNameAnyRecord:    AnyRecordType,
		QNameAnyGDoc:      AnyGDocType,
		QNameAnyCDoc:      AnyCDocType,
		QNameAnyWDoc:      AnyWDocType,
		QNameAnyODoc:      AnyODocType,
		QNameAnyObject:    AnyObjectType,
		QNameAnySingleton: AnySingletonType,
		QNameAnyView:      AnyViewType,
		QNameAnyExtension: AnyExtensionType,
		QNameAnyFunction:  AnyFunctionType,
		QNameAnyCommand:   AnyCommandType,
		QNameAnyQuery:     AnyQueryType,
	}
)

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
