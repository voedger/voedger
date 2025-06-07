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
//   - appdef.IFunction
type Function struct {
	Extension
	par, res types.TypeRef
}

func MakeFunc(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Function {
	return Function{Extension: MakeExtension(ws, name, kind)}
}

func (f Function) Param() appdef.IType { return f.par.Target(f.App().Type) }

func (f Function) Result() appdef.IType { return f.res.Target(f.App().Type) }

func (f *Function) setParam(name appdef.QName) { f.par.SetName(name) }

func (f *Function) setResult(name appdef.QName) { f.res.SetName(name) }

// Validates function
//
// # Returns error:
//   - if parameter type is unknown or not a Data, ODoc or Object,
//   - if result type is unknown or not a Data, Doc or Object,
func (f *Function) Validate() (err error) {
	if ok, e := f.par.Valid(f.App().Type); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown parameter type: %w", f, e))
	} else if typ := f.Param(); typ != nil {
		switch typ.Kind() {
		case appdef.TypeKind_Any, appdef.TypeKind_Data, appdef.TypeKind_ODoc, appdef.TypeKind_Object: // ok
		default:
			err = errors.Join(err, appdef.ErrInvalid("parameter type «%v», should be ODoc, Object or Data", typ))
		}
	}

	if ok, e := f.res.Valid(f.App().Type); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown result type: %w", f, e))
	} else if typ := f.Result(); typ != nil {
		switch typ.Kind() {
		case appdef.TypeKind_Any: // ok
		case appdef.TypeKind_Data, appdef.TypeKind_GDoc, appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_ODoc, appdef.TypeKind_Object: // ok
		default:
			err = errors.Join(err, appdef.ErrInvalid("result type «%v», should be Document, Object or Data", typ))
		}
	}

	return err
}

type FunctionBuilder struct {
	ExtensionBuilder
	*Function
}

func MakeFunctionBuilder(f *Function) FunctionBuilder {
	return FunctionBuilder{
		ExtensionBuilder: MakeExtensionBuilder(&f.Extension),
		Function:         f,
	}
}

func (fb *FunctionBuilder) SetParam(name appdef.QName) appdef.IFunctionBuilder {
	fb.Function.setParam(name)
	return fb
}

func (fb *FunctionBuilder) SetResult(name appdef.QName) appdef.IFunctionBuilder {
	fb.Function.setResult(name)
	return fb
}
