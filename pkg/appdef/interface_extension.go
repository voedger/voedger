/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Extension engine kind enumeration.
type ExtensionEngineKind uint8

//go:generate stringer -type=ExtensionEngineKind -output=stringer_extensionenginekind.go

const (
	ExtensionEngineKind_null ExtensionEngineKind = iota
	ExtensionEngineKind_BuiltIn
	ExtensionEngineKind_WASM

	ExtensionEngineKind_count
)

// Entry point is a type that can be executed.
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

	// Returns states.
	//
	// States are used to retrieve data.
	States() IStorages

	// Returns intents.
	//
	// Intents are used to store data.
	Intents() IStorages
}

type IExtensionBuilder interface {
	ITypeBuilder

	// Sets name.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetName(string) IExtensionBuilder

	// Sets engine.
	SetEngine(ExtensionEngineKind) IExtensionBuilder

	// Returns states builder.
	States() IStoragesBuilder

	// Returns intents builder.
	Intents() IStoragesBuilder
}
