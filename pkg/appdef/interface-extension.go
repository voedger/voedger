/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Extension engine kind enumeration.
//
// Ref. to extension-engine-kind.go for constants and methods
type ExtensionEngineKind uint8

// Entry point is a type that can be executed.
//
// Ref. to extension.go for implementation
type IExtension interface {
	IType

	// Extension entry point name.
	//
	// After construction new extension has a default name from type QName entity.
	Name() string

	// Engine kind.
	//
	// After construction new extension has a default BuiltIn engine.
	Engine() ExtensionEngineKind
}

type IExtensionBuilder interface {
	IExtension
	ITypeBuilder

	// Sets name.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetName(string) IExtensionBuilder

	// Sets engine.
	SetEngine(ExtensionEngineKind) IExtensionBuilder
}
