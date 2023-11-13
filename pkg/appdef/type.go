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

func (t *typ) IsSystem() bool {
	return t.QName().Pkg() == SysPackage
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

type typeRef struct {
	name QName
	t    IType
}

// Returns type by reference.
//
// If type is not found then returns nil.
func (r *typeRef) target(app IAppDef) IType {
	if r.name == NullQName {
		return nil
	}
	if r.name == QNameANY {
		return app.SysAny()
	}
	if (r.t == nil) || (r.t.QName() != r.name) {
		r.t = app.TypeByName(r.name)
	}
	return r.t
}

// Sets reference name
func (r *typeRef) setName(n QName) {
	r.name = n
	r.t = nil
}

// Returns is reference valid
func (r *typeRef) valid(app IAppDef) (bool, error) {
	if (r.name == NullQName) || (r.name == QNameANY) || (r.target(app) != nil) {
		return true, nil
	}
	return false, fmt.Errorf("type «%v» is not found: %w", r.name, ErrNameNotFound)
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
func (t *nullType) IsSystem() bool { return false }
func (t *nullType) Kind() TypeKind { return TypeKind_null }
func (t *nullType) QName() QName   { return NullQName }
func (t *nullType) String() string { return nullTypeString }

// AnyType is used for return then type is any
const anyTypeString = "any type"

type anyType struct {
	nullComment
	app IAppDef
}

func (t *anyType) App() IAppDef   { return t.app }
func (t *anyType) IsSystem() bool { return true }
func (t *anyType) Kind() TypeKind { return TypeKind_Any }
func (t *anyType) QName() QName   { return QNameANY }
func (t *anyType) String() string { return anyTypeString }
