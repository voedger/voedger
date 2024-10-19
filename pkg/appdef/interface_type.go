/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Types kinds enumeration
type TypeKind uint8

//go:generate stringer -type=TypeKind -output=stringer_typekind.go

const (
	TypeKind_null TypeKind = iota

	// Any type.
	//
	// Used as result types kind for functions that has parameter or result of any type.
	TypeKind_Any

	// Simple data types, like string, number, date, etc.
	TypeKind_Data

	// Global configuration, WSID==0 (global doc): UserProfileLocation, SystemConfig
	TypeKind_GDoc

	// Config document (per workspace articles, prices, clients)
	TypeKind_CDoc

	// Operational documents: bills, orders
	// https://vocable.ru/termin/operacionnyi-dokument.html
	// THE OPERATIONAL DOCUMENT: an invoice, receipt, order indicating the completion of the transaction
	// Might not be edited
	TypeKind_ODoc

	// bill
	// Workflow document, extends ODoc
	// Might be edited
	TypeKind_WDoc

	// Parts of documents, article_price, bill_item
	TypeKind_GRecord
	TypeKind_CRecord
	TypeKind_ORecord
	TypeKind_WRecord

	// collection (BO)  ((wsid, qname), id), record
	// logins ((wsid0), login) id
	TypeKind_ViewRecord

	// Function params, results, Event.command (this is command function params)
	TypeKind_Object

	// Functions
	TypeKind_Query
	TypeKind_Command
	TypeKind_Projector
	TypeKind_Job

	TypeKind_Workspace

	// Roles and grants
	TypeKind_Role

	// Rates and limits
	TypeKind_Rate
	TypeKind_Limit

	TypeKind_count
)

// # Type
//
// Type describes the entity, such as document, record or view.
type IType interface {
	IWithComments

	// Parent cache
	App() IAppDef

	// Type qualified name.
	QName() QName

	// Type kind
	Kind() TypeKind

	// Returns is type from system package.
	IsSystem() bool
}

// Interface describes the entity with types.
type IWithTypes interface {
	// Returns type by name.
	//
	// If not found then empty type with TypeKind_null is returned
	Type(name QName) IType

	// Returns type by name.
	//
	// Returns nil if type not found.
	TypeByName(name QName) IType

	// Enumerates all internal types.
	//
	// Types are enumerated in alphabetical order of QNames.
	Types(func(IType) bool)
}

type ITypeBuilder interface {
	ICommentsBuilder
}

// AnyType is used for return then type is any
var AnyType = newAnyType(QNameANY)

// Any×××Type are used for substitution, e.g. for rate limits, projector events, etc.
var (
	AnyStructureType = newAnyType(QNameAnyStructure)
	AnyRecordType    = newAnyType(QNameAnyRecord)
	AnyGDocType      = newAnyType(QNameAnyGDoc)
	AnyCDocType      = newAnyType(QNameAnyCDoc)
	AnyWDocType      = newAnyType(QNameAnyWDoc)
	AnySingletonType = newAnyType(QNameAnySingleton)
	AnyODocType      = newAnyType(QNameAnyODoc)
	AnyObjectType    = newAnyType(QNameAnyObject)
	AnyViewType      = newAnyType(QNameAnyView)
	AnyExtensionType = newAnyType(QNameAnyExtension)
	AnyFunctionType  = newAnyType(QNameAnyFunction)
	AnyCommandType   = newAnyType(QNameAnyCommand)
	AnyQueryType     = newAnyType(QNameAnyQuery)
)
