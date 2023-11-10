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
	typ
	parent   interface{}
	par, res typeRef
	ext      *extension
}

func makeFunc(app *appDef, name QName, kind TypeKind, parent interface{}) function {
	f := function{
		typ:    makeType(app, name, kind),
		parent: parent,
		ext:    newExtension(),
	}
	return f
}

func (f *function) Param() IType {
	return f.par.target(f.app)
}

func (f *function) Extension() IExtension {
	return f.ext
}

func (f *function) Result() IType {
	return f.res.target(f.app)
}

func (f *function) SetParam(name QName) IFunctionBuilder {
	f.par.setName(name)
	return f.parent.(IFunctionBuilder)
}

func (f *function) SetResult(name QName) IFunctionBuilder {
	f.res.setName(name)
	return f.parent.(IFunctionBuilder)
}

func (f *function) SetExtension(name string, engine ExtensionEngineKind, comment ...string) IFunctionBuilder {
	if name == "" {
		panic(fmt.Errorf("%v: extension name is empty: %w", f, ErrNameMissed))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", f, name, err))
	}
	f.ext.name = name
	f.ext.engine = engine
	f.ext.SetComment(comment...)
	return f
}

// Validates function
func (f *function) Validate() (err error) {
	if ok, e := f.par.valid(f.app); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown parameter type: %w", f, e))
	}

	if ok, e := f.res.valid(f.app); !ok {
		err = errors.Join(err, fmt.Errorf("%v: invalid or unknown result type: %w", f, e))
	}

	if f.Extension().Name() == "" {
		err = errors.Join(err, fmt.Errorf("%v: command extension name is missed: %w", f, ErrNameMissed))
	}

	if f.Extension().Engine() == ExtensionEngineKind_null {
		err = errors.Join(err, fmt.Errorf("%v: command extension engine is missed: %w", f, ErrExtensionEngineKindMissed))
	}

	return err
}
