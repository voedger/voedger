/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// # Implements:
//   - IType
type typ struct {
	comment
	app  *appDef
	ws   *workspace
	name QName
	kind TypeKind
}

// Creates and returns new type.
//
// Name can be empty (NullQName), then type is anonymous.
func makeType(app *appDef, ws *workspace, name QName, kind TypeKind) typ {
	if name != NullQName {
		if ok, err := ValidQName(name); !ok {
			panic(fmt.Errorf("invalid type name «%v»: %w", name, err))
		}
	}
	return typ{comment{}, app, ws, name, kind}
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

func (t *typ) Workspace() IWorkspace {
	return t.ws
}

// # Implements:
//   - ITypeBuilder
type typeBuilder struct {
	commentBuilder
	*typ
}

func makeTypeBuilder(typ *typ) typeBuilder {
	return typeBuilder{
		commentBuilder: makeCommentBuilder(&typ.comment),
		typ:            typ,
	}
}

func (t *typeBuilder) String() string { return t.typ.String() }

type typeRef struct {
	name QName
	typ  IType
}

// List of types.
type types[T IType] struct {
	m map[QName]T
	s []T
}

// Creates and returns new types.
func newTypes[T IType]() *types[T] {
	return &types[T]{m: make(map[QName]T)}
}

func (tt *types[T]) add(t T) {
	name := t.QName()

	tt.m[name] = t
	tt.s = nil
}

func (tt *types[T]) all(visit func(T) bool) {
	if len(tt.s) != len(tt.m) {
		tt.s = slices.SortedFunc(maps.Values(tt.m), func(i, j T) int {
			return CompareQName(i.QName(), j.QName())
		})
	}
	slices.Values(tt.s)(visit)
}

func (tt *types[T]) clear() {
	tt.m = make(map[QName]T)
	tt.s = nil
}

func (tt types[T]) find(name QName) IType {
	if t, ok := tt.m[name]; ok {
		return t
	}
	return NullType
}

// Returns type by reference.
//
// If type is not found then returns nil.
func (r *typeRef) target(tt FindType) IType {
	if r.name == NullQName {
		return nil
	}
	if r.name == QNameANY {
		return AnyType
	}
	if (r.typ == nil) || (r.typ.QName() != r.name) {
		r.typ = nil
		if t := tt(r.name); t.Kind() != TypeKind_null {
			r.typ = t
		}
	}
	return r.typ
}

// Sets reference name
func (r *typeRef) setName(n QName) {
	r.name = n
	r.typ = nil
}

// Returns is reference valid
func (r *typeRef) valid(tt FindType) (bool, error) {
	if (r.name == NullQName) || (r.name == QNameANY) || (r.target(tt) != nil) {
		return true, nil
	}
	return false, ErrTypeNotFound(r.name)
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

const nullTypeString = "null type"

type nullType struct{ nullComment }

func (t nullType) App() IAppDef          { return nil }
func (t nullType) IsSystem() bool        { return false }
func (t nullType) Kind() TypeKind        { return TypeKind_null }
func (t nullType) QName() QName          { return NullQName }
func (t nullType) String() string        { return nullTypeString }
func (t nullType) Workspace() IWorkspace { return nil }

type anyType struct {
	nullComment
	name QName
}

func newAnyType(name QName) IType { return &anyType{nullComment{}, name} }

func (t anyType) App() IAppDef          { return nil }
func (t anyType) IsSystem() bool        { return true }
func (t anyType) Kind() TypeKind        { return TypeKind_Any }
func (t anyType) QName() QName          { return t.name }
func (t anyType) String() string        { return t.name.Entity() + " type" }
func (t anyType) Workspace() IWorkspace { return nil }
