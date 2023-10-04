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

// Command
//
// Ref. to command.go for implementation
type ICommand interface {
	IType

	// Argument. Returns nil if not assigned
	Arg() IObject

	// Unlogged (secure) argument. Returns nil if not assigned
	UnloggedArg() IObject

	// Result. Returns nil if not assigned
	Result() IObject

	// Extension
	Extension() IExtension
}

type ICommandBuilder interface {
	ICommand
	ICommentBuilder

	// Sets command argument. Must be object or NullQName
	SetArg(QName) ICommandBuilder

	// Sets command unlogged (secure) argument. Must be object or NullQName
	SetUnloggedArg(QName) ICommandBuilder

	// Sets command result. Must be object or NullQName
	SetResult(QName) ICommandBuilder

	// Sets engine.
	//
	// # Panics:
	//	- if name is empty or invalid identifier
	SetExtension(name string, engine ExtensionEngineKind, comment ...string) ICommandBuilder
}
