/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Command is a function that changes system state.
// Command may have unlogged argument.
// Unlogged argument is a secure argument that is not logged.
//
// Ref. to command.go for implementation
type ICommand interface {
	IFunc

	// Unlogged (secure) argument. Returns nil if not assigned
	UnloggedArg() IObject
}

type ICommandBuilder interface {
	ICommand
	IFuncBuilder

	// Sets command unlogged (secure) argument. Must be object or NullQName
	SetUnloggedArg(QName) ICommandBuilder
}
