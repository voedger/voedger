/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//	- ICommand & ICommandBuilder
type command struct {
	def
	arg, unl, res objRef
	ext           *extension
}

func newCommand(app *appDef, name QName) *command {
	cmd := &command{
		def: makeDef(app, name, DefKind_Command),
		arg: objRef{NullQName, NullObject},
		unl: objRef{NullQName, NullObject},
		res: objRef{NullQName, NullObject},
		ext: newExtension(),
	}
	app.appendDef(cmd)
	return cmd
}

func (cmd *command) Arg() IObject {
	return cmd.arg.object(cmd.app)
}

func (cmd *command) Extension() IExtension {
	return cmd.ext
}

func (cmd *command) Result() IObject {
	return cmd.res.object(cmd.app)
}

func (cmd *command) SetArg(name QName) ICommandBuilder {
	cmd.arg.name = name
	return cmd
}

func (cmd *command) SetUnloggedArg(name QName) ICommandBuilder {
	cmd.unl.name = name
	return cmd
}

func (cmd *command) SetResult(name QName) ICommandBuilder {
	cmd.res.name = name
	return cmd
}

func (cmd *command) SetExtension(name string, engine ExtensionEngineKind) ICommandBuilder {
	cmd.ext.name = name
	cmd.ext.engine = engine
	return cmd
}

func (cmd *command) UnloggedArg() IObject {
	return cmd.unl.object(cmd.app)
}
