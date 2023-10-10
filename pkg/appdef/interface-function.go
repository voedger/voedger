/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Extension engine kind enumeration.
//
// Ref. to extension-engine-kind.go for constants and methods
type ExtensionEngineKind uint8

// Entry point for extension
//
// Ref. to extension.go for implementation
type IExtension interface {
	IComment

	// Extension entry point name
	Name() string

	// Engine kind
	Engine() ExtensionEngineKind
}

// Function is abstract to inherit commands and queries
//
// Ref. to function.go for implementation
type IFunc interface {
	IType

	// Argument. Returns nil if not assigned
	Arg() IObject

	// Result. Returns nil if not assigned
	Result() IObject

	// Extension
	Extension() IExtension
}

type IFuncBuilder interface {
	IFunc
	ITypeBuilder

	// Sets command argument. Must be object or NullQName
	SetArg(QName) IFuncBuilder

	// Sets command result. Must be object or NullQName
	SetResult(QName) IFuncBuilder

	// Sets engine.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetExtension(name string, engine ExtensionEngineKind, comment ...string) IFuncBuilder
}
