/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// # Implements:
//   - IType
//   - ITypeBuilder
type typ struct {
	comment
	app  *appDef
	name QName
	kind TypeKind
}

// Creates and returns new type.
//
// Name can be empty (NullQName), then type is anonymous.
func makeType(app *appDef, name QName, kind TypeKind) typ {
	if name != NullQName {
		if ok, err := ValidQName(name); !ok {
			panic(fmt.Errorf("invalid type name «%v»: %w", name, err))
		}
	}
	return typ{comment{}, app, name, kind}
}

func (t *typ) App() IAppDef {
	return t.app
}

func (t *typ) Kind() TypeKind {
	return t.kind
}

func (t *typ) QName() QName {
	return t.name
}

func (t *typ) String() string {
	return fmt.Sprintf("%s «%v»", t.Kind().TrimString(), t.QName())
}

// Validate specified type.
//
// # Validation:
//   - if type supports Validate() interface, then call this,
//   - if structured type has fields, validate fields,
//   - if structured type has containers, validate containers
func validateType(t IType) (err error) {
	if v, ok := t.(interface{ Validate() error }); ok {
		err = v.Validate()
	}

	if _, ok := t.(IFields); ok {
		err = errors.Join(err, validateTypeFields(t))
	}

	if _, ok := t.(IContainers); ok {
		err = errors.Join(err, validateTypeContainers(t))
	}

	return err
}

// NullType is used for return then type is not founded
const nullTypeString = "null type"

var NullType = new(nullType)

type nullType struct{ nullComment }

func (t *nullType) App() IAppDef   { return nil }
func (t *nullType) Kind() TypeKind { return TypeKind_null }
func (t *nullType) QName() QName   { return NullQName }
func (t *nullType) String() string { return nullTypeString }
