/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Command is a function that changes system state.
// Command may have unlogged parameter.
// Unlogged parameter is a secure parameter that is not logged.
type ICommand interface {
	IFunction

	// Unlogged (secure) parameter. Returns nil if not assigned
	UnloggedParam() IType
}

type ICommandBuilder interface {
	ICommand
	IFunctionBuilder

	// Sets command unlogged (secure) parameter. Must be known type from next kinds:
	//	 - Data
	//	 - ODoc
	//	 - Object
	//
	// If NullQName passed then it means that command has no unlogged parameter.
	// If QNameANY passed then it means that command unlogged parameter may be any.
	SetUnloggedParam(QName) ICommandBuilder
}
