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
//
// Ref to qname.go for constants and methods
type QName struct {
	pkg    string
	entity string
}

// Types kinds enumeration.
//
// Ref. type-kind.go for constants and methods
type TypeKind uint8

// # Type
//
// Type describes the entity, such as document, record or view.
//
// Ref to type.go for implementation
type IType interface {
	IComment

	// Parent cache
	App() IAppDef

	// Type qualified name.
	QName() QName

	// Type kind
	Kind() TypeKind
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
	IType
	ICommentBuilder
}
