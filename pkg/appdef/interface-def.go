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

// Definition kind enumeration.
//
// Ref. def-kind.go for constants and methods
type DefKind uint8

// Definition describes the entity, such as document, record or view. Definitions may have fields and containers.
//
// Ref to def.go for implementation
type IDef interface {
	// Parent cache
	App() IAppDef

	// Definition qualified name
	QName() QName

	// Definition kind.
	Kind() DefKind
}
