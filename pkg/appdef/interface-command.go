/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Command is a function that changes system state.
// Command may have unlogged parameter.
// Unlogged parameter is a secure parameter that is not logged.
//
// Ref. to command.go for implementation
type ICommand interface {
	IFunction

	// Unlogged (secure) parameter. Returns nil if not assigned
	UnloggedParam() IType
}

type ICommandBuilder interface {
	ICommand
	IFunctionBuilder

	// Sets command unlogged (secure) parameter. Must be known structural or data type.
	// If NullQName passed then it means that command has no unlogged parameter.
	// If QNameAny passed then it means that command unlogged parameter may be any structure or data type.
	SetUnloggedParam(QName) ICommandBuilder
}
