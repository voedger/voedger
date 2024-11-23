/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"github.com/voedger/voedger/pkg/goutils/set"
)

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

type TypeKindSet = set.Set[TypeKind]

var (
	TypeKind_Docs = func() TypeKindSet {
		s := set.From(
			TypeKind_GDoc,
			TypeKind_CDoc,
			TypeKind_ODoc,
			TypeKind_WDoc,
		)
		s.SetReadOnly()
		return s
	}()

	TypeKind_Records = func() TypeKindSet {
		s := set.From(TypeKind_Docs.AsArray()...)
		s.Set(
			TypeKind_GRecord,
			TypeKind_CRecord,
			TypeKind_ORecord,
			TypeKind_WRecord,
		)
		s.SetReadOnly()
		return s
	}()

	TypeKind_Structures = func() TypeKindSet {
		s := set.From(TypeKind_Records.AsArray()...)
		s.Set(TypeKind_Object)
		s.SetReadOnly()
		return s
	}()

	TypeKind_Singletons = func() TypeKindSet {
		s := set.From(
			TypeKind_CDoc,
			TypeKind_WDoc,
		)
		s.SetReadOnly()
		return s
	}()

	TypeKind_Functions = func() TypeKindSet {
		s := set.From(
			TypeKind_Command,
			TypeKind_Query,
		)
		s.SetReadOnly()
		return s
	}()

	TypeKind_Extensions = func() TypeKindSet {
		s := set.From(TypeKind_Functions.AsArray()...)
		s.Set(
			TypeKind_Projector,
			TypeKind_Job,
		)
		s.SetReadOnly()
		return s
	}()
)

// # Type
//
// Type describes the entity, such as document, record or view.
type IType interface {
	IWithComments

	// Application
	App() IAppDef

	// Workspace
	Workspace() IWorkspace

	// Type qualified name.
	QName() QName

	// Type kind
	Kind() TypeKind

	// Returns is type from system package.
	IsSystem() bool
}

type (
	// Finds type by name.
	//
	// If not found then empty type with TypeKind_null is returned
	FindType func(QName) IType

	// Types iterator.
	SeqType func(func(IType) bool)
)

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
