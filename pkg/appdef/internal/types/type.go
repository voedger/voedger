/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package types

import (
	"fmt"
	"iter"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/slicex"
)

// # Supports:
//   - appdef.IType
type Typ struct {
	comments.WithComments
	WithTags
	app  appdef.IAppDef
	ws   appdef.IWorkspace
	name appdef.QName
	kind appdef.TypeKind
}

// Creates and returns new type.
//
// Name can be empty (appdef.NullQName), then type is anonymous.
func MakeType(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Typ {
	if name != appdef.NullQName {
		if ok, err := appdef.ValidQName(name); !ok {
			panic(fmt.Errorf("invalid type name «%v»: %w", name, err))
		}
		if app.Type(name).Kind() != appdef.TypeKind_null {
			panic(appdef.ErrAlreadyExists("type «%v»", name))
		}
	}
	find := app.Type
	if ws != nil {
		find = ws.LocalType // #2889 $VSQL_TagNonExp: only local tags can be used
	}
	t := Typ{
		WithComments: comments.MakeWithComments(),
		WithTags:     MakeWithTags(find), // #2889 $VSQL_TagNonExp: only local tags can be used
		app:          app,
		ws:           ws,
		name:         name,
		kind:         kind,
	}
	return t
}

func (t Typ) App() appdef.IAppDef { return t.app }

func (t Typ) IsSystem() bool {
	return t.QName().Pkg() == appdef.SysPackage
}

func (t Typ) Kind() appdef.TypeKind { return t.kind }

func (t Typ) QName() appdef.QName { return t.name }

func (t Typ) String() string {
	return fmt.Sprintf("%s «%v»", t.Kind().TrimString(), t.QName())
}

func (t Typ) Workspace() appdef.IWorkspace { return t.ws }

// # Supports:
//   - appdef.ITypeBuilder
type TypeBuilder struct {
	comments.CommentBuilder
	TagBuilder
	*Typ
}

func MakeTypeBuilder(t *Typ) TypeBuilder {
	return TypeBuilder{
		CommentBuilder: comments.MakeCommentBuilder(&t.WithComments),
		TagBuilder:     MakeTagBuilder(&t.WithTags),
		Typ:            t,
	}
}

func (t *TypeBuilder) String() string { return t.Typ.String() }

type TypeRef struct {
	name appdef.QName
	typ  appdef.IType
}

// Returns referenced type name
func (r TypeRef) Name() appdef.QName { return r.name }

// Returns type by reference.
//
// If type is not found then returns nil.
func (r TypeRef) Target(find appdef.FindType) appdef.IType {
	switch r.name {
	case appdef.NullQName:
		return nil
	case appdef.QNameANY:
		return appdef.AnyType
	default:
		if (r.typ != nil) && (r.typ.QName() == r.name) {
			return r.typ
		}
	}
	if t := find(r.name); t.Kind() != appdef.TypeKind_null {
		return t
	}
	return nil
}

// Sets referenced type name
func (r *TypeRef) SetName(n appdef.QName) {
	r.name = n
	r.typ = nil
}

// Returns is reference valid
func (r *TypeRef) Valid(find appdef.FindType) (bool, error) {
	if (r.name == appdef.NullQName) || (r.name == appdef.QNameANY) {
		return true, nil
	}
	if t := r.Target(find); t != nil {
		if r.typ != t {
			r.typ = t
		}
		return true, nil
	}

	return false, appdef.ErrTypeNotFound(r.name)
}

// List of Types.
type Types[T appdef.IType] struct {
	m map[appdef.QName]T
	s []T
}

// Creates and returns new types.
func NewTypes[T appdef.IType]() *Types[T] {
	return &Types[T]{m: make(map[appdef.QName]T)}
}

func (tt *Types[T]) Add(t T) {
	tt.m[t.QName()] = t
	tt.s = slicex.InsertInSort(tt.s, t, func(t1, t2 T) int { return appdef.CompareQName(t1.QName(), t2.QName()) })
}

func (tt *Types[T]) Clear() {
	tt.m = make(map[appdef.QName]T)
	tt.s = nil
}

func (tt Types[T]) Find(name appdef.QName) appdef.IType {
	if t, ok := tt.m[name]; ok {
		return t
	}
	return appdef.NullType
}

func (tt Types[T]) Values() iter.Seq[T] { return slices.Values(tt.s) }

const nullTypeString = "null type"

// # Supports
//   - appdef.IType
type NullType struct {
	comments.NullComment
	NullTags
}

func (t NullType) App() appdef.IAppDef          { return nil }
func (t NullType) IsSystem() bool               { return false }
func (t NullType) Kind() appdef.TypeKind        { return appdef.TypeKind_null }
func (t NullType) QName() appdef.QName          { return appdef.NullQName }
func (t NullType) String() string               { return nullTypeString }
func (t NullType) Workspace() appdef.IWorkspace { return nil }
