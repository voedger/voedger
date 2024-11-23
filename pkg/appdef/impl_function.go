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
//   - IFunction
//   - IFuncBuilder
type function struct {
	extension
	par, res typeRef
}

func makeFunc(app *appDef, ws *workspace, name QName, kind TypeKind) function {
	f := function{
		extension: makeExtension(app, ws, name, kind),
	}
	return f
}

func (f *function) Param() IType { return f.par.target(f.app.Type) }

func (f *function) Result() IType { return f.res.target(f.app.Type) }

func (f *function) setParam(name QName) { f.par.setName(name) }

func (f *function) setResult(name QName) { f.res.setName(name) }

// Validates function
//
// # Returns error:
//   - if parameter type is unknown or not a Data, ODoc or Object,
//   - if result type is unknown or not a Data, Doc or Object,
func (f *function) Validate() (err error) {
	if ok, e := f.par.valid(f.app.Type); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown parameter type: %w", f, e))
	} else if typ := f.Param(); typ != nil {
		switch typ.Kind() {
		case TypeKind_Any, TypeKind_Data, TypeKind_ODoc, TypeKind_Object: // ok
		default:
			err = errors.Join(err, ErrInvalid("parameter type «%v», should be ODoc, Object or Data", typ))
		}
	}

	if ok, e := f.res.valid(f.app.Type); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown result type: %w", f, e))
	} else if typ := f.Result(); typ != nil {
		switch typ.Kind() {
		case TypeKind_Any: // ok
		case TypeKind_Data, TypeKind_GDoc, TypeKind_CDoc, TypeKind_WDoc, TypeKind_ODoc, TypeKind_Object: // ok
		default:
			err = errors.Join(err, ErrInvalid("result type «%v», should be Document, Object or Data", typ))
		}
	}

	return err
}

type functionBuilder struct {
	extensionBuilder
	*function
}

func makeFunctionBuilder(f *function) functionBuilder {
	return functionBuilder{
		extensionBuilder: makeExtensionBuilder(&f.extension),
		function:         f,
	}
}

func (fb *functionBuilder) SetParam(name QName) IFunctionBuilder {
	fb.function.setParam(name)
	return fb
}

func (fb *functionBuilder) SetResult(name QName) IFunctionBuilder {
	fb.function.setResult(name)
	return fb
}
