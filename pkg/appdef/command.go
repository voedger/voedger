/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// Command
//
// # Implements:
//   - ICommand
//   - ICommandBuilder
type command struct {
	function
	unl objRef
}

func newCommand(app *appDef, name QName) *command {
	cmd := &command{}
	cmd.function = makeFunc(app, name, TypeKind_Command, cmd)
	app.appendType(cmd)
	return cmd
}

func (cmd *command) SetUnloggedParam(name QName) ICommandBuilder {
	cmd.unl.setName(name)
	return cmd
}

func (cmd *command) UnloggedParam() IObject {
	return cmd.unl.object(cmd.app)
}

// Validates command
func (cmd *command) Validate() (err error) {
	err = cmd.function.Validate()

	if cmd.unl.name != NullQName {
		if cmd.unl.object(cmd.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: unlogged object type «%v» is not found: %w", cmd.QName(), cmd.unl.name, ErrNameNotFound))
		}
	}

	return err
}
