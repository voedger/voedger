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

// Function is a type of object that can be called.
// Function may have argument and result.
// Function may have extension.
// Function may be query or command.
//
// Ref. to function.go for implementation
type IFunction interface {
	IType

	// Argument. Returns nil if not assigned
	Arg() IObject

	// Result. Returns nil if not assigned
	Result() IObject

	// Extension
	Extension() IExtension
}

type IFunctionBuilder interface {
	IFunction
	ITypeBuilder

	// Sets command argument. Must be object or NullQName
	SetArg(QName) IFunctionBuilder

	// Sets command result. Must be object or NullQName
	SetResult(QName) IFunctionBuilder

	// Sets engine.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetExtension(name string, engine ExtensionEngineKind, comment ...string) IFunctionBuilder
}
