/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Command is a function
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
