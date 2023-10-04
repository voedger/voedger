/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Query
//
// Ref. to query.go for implementation
type IQuery interface {
	IType

	// Argument. Returns nil if not assigned
	Arg() IObject

	// Result. Returns nil if not assigned.
	//
	// If result is may be different, then NullQName is used
	Result() IObject

	// Extension
	Extension() IExtension
}

type IQueryBuilder interface {
	IQuery
	ICommentBuilder

	// Sets query argument. Must be object or NullQName
	SetArg(QName) IQueryBuilder

	// Sets query result. Must be object or NullQName
	SetResult(QName) IQueryBuilder

	// Sets engine.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetExtension(name string, engine ExtensionEngineKind) IQueryBuilder
}
