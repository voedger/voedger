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
	c := &Command{Function: MakeFunc(ws, name, appdef.TypeKind_Command)}
	types.Propagate(c)
	return c
}

func (c Command) UnloggedParam() appdef.IType {
	return c.unl.Target(c.App().Type)
}

// Validates command
func (c *Command) Validate() (err error) {
	err = c.Function.Validate()

	if ok, e := c.unl.Valid(c.App().Type); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown unlogged parameter type: %w", c, e))
	} else if typ := c.UnloggedParam(); typ != nil {
		switch typ.Kind() {
		case appdef.TypeKind_Any, appdef.TypeKind_Data, appdef.TypeKind_ODoc, appdef.TypeKind_Object: // ok
		default:
			err = errors.Join(err, appdef.ErrInvalid("unlogged parameter type «%v», should be ODoc, Object or Data", typ))
		}
	}

	return err
}

func (c *Command) setUnloggedParam(name appdef.QName) {
	c.unl.SetName(name)
}

type CommandBuilder struct {
	FunctionBuilder
	c *Command
}

func NewCommandBuilder(c *Command) *CommandBuilder {
	return &CommandBuilder{
		FunctionBuilder: MakeFunctionBuilder(&c.Function),
		c:               c,
	}
}

func (cb *CommandBuilder) SetUnloggedParam(name appdef.QName) appdef.ICommandBuilder {
	cb.c.setUnloggedParam(name)
	return cb
}
