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

// type IWithCommands interface {
// 	// Returns Command by name.
// 	//
// 	// Returns nil if not found.
// 	Command(QName) ICommand

// 	// Enumerates all commands
// 	//
// 	// Commands are enumerated in alphabetical order by QName
// 	Commands(func(ICommand) bool)
// }

type ICommandBuilder interface {
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

type ICommandsBuilder interface {
	// Adds new command.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddCommand(QName) ICommandBuilder
}
