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
//   - ICommand
//   - ICommandBuilder
type command struct {
	function
	unl typeRef
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

func (cmd *command) UnloggedParam() IType {
	return cmd.unl.target(cmd.app)
}

// Validates command
func (cmd *command) Validate() (err error) {
	err = cmd.function.Validate()

	if ok, e := cmd.unl.valid(cmd.app); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown unlogged parameter type: %w", cmd, e))
	}

	return err
}
