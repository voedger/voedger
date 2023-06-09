/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// # Implements:
//   - ICommand & ICommandBuilder
type command struct {
	def
	arg, unl, res objRef
	ext           extension
}

func newCommand(app *appDef, name QName) *command {
	cmd := &command{
		def: makeDef(app, name, DefKind_Command),
	}
	app.appendDef(cmd)
	return cmd
}

func (cmd *command) Arg() IObject {
	return cmd.arg.object(cmd.app)
}

func (cmd *command) Extension() IExtension {
	return &cmd.ext
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
	if name == "" {
		panic(fmt.Errorf("%v: extension name is empty: %w", cmd.QName(), ErrNameMissed))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", cmd.QName(), name, err))
	}
	cmd.ext.name = name
	cmd.ext.engine = engine
	return cmd
}

func (cmd *command) UnloggedArg() IObject {
	return cmd.unl.object(cmd.app)
}

// validates command
func (cmd *command) Validate() (err error) {
	if cmd.arg.name != NullQName {
		if cmd.arg.object(cmd.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: argument object «%v» is not found: %w", cmd.QName(), cmd.arg.name, ErrNameNotFound))
		}
	}

	if cmd.unl.name != NullQName {
		if cmd.unl.object(cmd.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: unlogged argument object «%v» is not found: %w", cmd.QName(), cmd.unl.name, ErrNameNotFound))
		}
	}

	if cmd.res.name != NullQName {
		if cmd.res.object(cmd.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: command result object «%v» is not found: %w", cmd.QName(), cmd.res.name, ErrNameNotFound))
		}
	}

	if cmd.Extension().Name() == "" {
		err = errors.Join(err, fmt.Errorf("%v: command extension name is missed: %w", cmd.QName(), ErrNameMissed))
	}

	if cmd.Extension().Engine() == ExtensionEngineKind_null {
		err = errors.Join(err, fmt.Errorf("%v: command extension engine is missed: %w", cmd.QName(), ErrExtensionEngineKindMissed))
	}

	return err
}
