/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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

// Type describes the entity with Types(func(IType)) menthod
type IWithTypes interface {
	// Enumerates all internal types.
	Types(func(IType))
}

type ITypeBuilder interface {
	IType
	ICommentBuilder
}
