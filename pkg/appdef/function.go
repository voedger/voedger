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
//   - IFunc
//   - IFuncBuilder
type function struct {
	typ
	parent   interface{}
	arg, res objRef
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

func (f *function) Arg() IObject {
	return f.arg.object(f.app)
}

func (f *function) Extension() IExtension {
	return f.ext
}

func (f *function) Result() IObject {
	return f.res.object(f.app)
}

func (f *function) SetArg(name QName) IFuncBuilder {
	f.arg.setName(name)
	return f.parent.(IFuncBuilder)
}

func (f *function) SetResult(name QName) IFuncBuilder {
	f.res.setName(name)
	return f.parent.(IFuncBuilder)
}

func (f *function) SetExtension(name string, engine ExtensionEngineKind, comment ...string) IFuncBuilder {
	if name == "" {
		panic(fmt.Errorf("%v: extension name is empty: %w", f.QName(), ErrNameMissed))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", f.QName(), name, err))
	}
	f.ext.name = name
	f.ext.engine = engine
	f.ext.SetComment(comment...)
	return f
}

// Validates function
func (f *function) Validate() (err error) {
	if f.arg.name != NullQName {
		if f.arg.object(f.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: argument type «%v» is not found: %w", f.QName(), f.arg.name, ErrNameNotFound))
		}
	}

	if f.res.name != NullQName {
		if f.res.object(f.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: command result type «%v» is not found: %w", f.QName(), f.res.name, ErrNameNotFound))
		}
	}

	if f.Extension().Name() == "" {
		err = errors.Join(err, fmt.Errorf("%v: command extension name is missed: %w", f.QName(), ErrNameMissed))
	}

	if f.Extension().Engine() == ExtensionEngineKind_null {
		err = errors.Join(err, fmt.Errorf("%v: command extension engine is missed: %w", f.QName(), ErrExtensionEngineKindMissed))
	}

	return err
}
