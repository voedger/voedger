/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # QName
//
// Qualified name
//
// <pkg>.<entity>
type QName struct {
	pkg    string
	entity string
}

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

	// Глобальный Global configuration, WSID==0 (глобальная номенклатура): UserProfileLocation, SystemConfig
	TypeKind_GDoc

	// Конфигурационный документ (per workspace articles, prices, clients)
	TypeKind_CDoc

	// Operational documents: bills, orders
	// https://vocable.ru/termin/operacionnyi-dokument.html
	// ОПЕРАЦИОННЫЙ ДОКУМЕНТ счет-фактура, чек, заказ, свидетельствующий о совершении сделки.
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

	TypeKind_Workspace

	TypeKind_FakeLast
)

// # Type
//
// Type describes the entity, such as document, record or view.
type IType interface {
	IComment

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
	Types(func(IType))
}

type ITypeBuilder interface {
	ICommentBuilder
}
