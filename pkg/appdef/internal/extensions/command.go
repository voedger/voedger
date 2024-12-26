/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.ICommand
type Command struct {
	Function
	unl types.TypeRef
}

func NewCommand(ws appdef.IWorkspace, name appdef.QName) *Command {
	return &Command{Function: MakeFunc(ws, name, appdef.TypeKind_Command)}
}

func (cmd *Command) UnloggedParam() appdef.IType {
	return cmd.unl.Target(cmd.App().Type)
}

// Validates command
func (cmd *Command) Validate() (err error) {
	err = cmd.Function.Validate()

	if ok, e := cmd.unl.Valid(cmd.App().Type); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown unlogged parameter type: %w", cmd, e))
	} else if typ := cmd.UnloggedParam(); typ != nil {
		switch typ.Kind() {
		case appdef.TypeKind_Any, appdef.TypeKind_Data, appdef.TypeKind_ODoc, appdef.TypeKind_Object: // ok
		default:
			err = errors.Join(err, appdef.ErrInvalid("unlogged parameter type «%v», should be ODoc, Object or Data", typ))
		}
	}

	return err
}

func (cmd *Command) setUnloggedParam(name appdef.QName) {
	cmd.unl.SetName(name)
}

type CommandBuilder struct {
	FunctionBuilder
	*Command
}

func NewCommandBuilder(command *Command) *CommandBuilder {
	return &CommandBuilder{
		FunctionBuilder: MakeFunctionBuilder(&command.Function),
		Command:         command,
	}
}

func (cb *CommandBuilder) SetUnloggedParam(name appdef.QName) appdef.ICommandBuilder {
	cb.Command.setUnloggedParam(name)
	return cb
}
