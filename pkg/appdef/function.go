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
	embeds   interface{}
	par, res typeRef
}

func makeFunc(app *appDef, name QName, kind TypeKind, embeds interface{}) function {
	f := function{
		extension: makeExtension(app, name, kind, embeds),
		embeds:    embeds,
	}
	return f
}

func (f *function) Param() IType {
	return f.par.target(f.app)
}

func (f *function) Result() IType {
	return f.res.target(f.app)
}

func (f *function) SetParam(name QName) IFunctionBuilder {
	f.par.setName(name)
	return f.embeds.(IFunctionBuilder)
}

func (f *function) SetResult(name QName) IFunctionBuilder {
	f.res.setName(name)
	return f.embeds.(IFunctionBuilder)
}

// Validates function
func (f *function) Validate() (err error) {
	if ok, e := f.par.valid(f.app); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown parameter type: %w", f, e))
	} else if typ := f.Param(); typ != nil {
		switch typ.Kind() {
		case TypeKind_Any: // ok
		case TypeKind_Data, TypeKind_ODoc, TypeKind_Object: // ok
		default:
			err = errors.Join(err, fmt.Errorf("%v: parameter type is %v, must be ODoc, Object or Data: %w", f, typ, ErrInvalidTypeKind))
		}
	}

	if ok, e := f.res.valid(f.app); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown result type: %w", f, e))
	} else if typ := f.Result(); typ != nil {
		switch typ.Kind() {
		case TypeKind_Any: // ok
		case TypeKind_Data, TypeKind_GDoc, TypeKind_CDoc, TypeKind_WDoc, TypeKind_ODoc, TypeKind_Object: // ok
		default:
			err = errors.Join(err, fmt.Errorf("%v: result type is %v, must be Document, Object or Data: %w", f, typ, ErrInvalidTypeKind))
		}
	}

	return err
}
