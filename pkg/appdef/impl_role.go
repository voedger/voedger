/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// # Implements:
//   - IRole
type role struct {
	typ
	grants []*grant
}

func newRole(app *appDef, name QName) *role {
	r := &role{
		typ:    makeType(app, name, TypeKind_Role),
		grants: make([]*grant, 0),
	}
	app.appendType(r)
	return r
}

func (r role) Grants(cb func(IGrant)) {
	for _, g := range r.grants {
		cb(g)
	}
}

func (r role) GrantsByKind(k GrantKind, cb func(IGrant)) {
	for _, g := range r.grants {
		if g.Kind() == k {
			cb(g)
		}
	}
}

func (r role) GrantsForObject(name QName) []IGrant {
	gg := make([]IGrant, 0)
	for _, g := range r.grants {
		if g.Objects().Contains(name) {
			gg = append(gg, g)
		}
	}
	return gg
}

// appends a grant to the role
func (r *role) appendGrant(g *grant) {
	r.grants = append(r.grants, g)
}

// # Implements:
//   - IRoleBuilder
type roleBuilder struct {
	typeBuilder
	*role
}

func newRoleBuilder(role *role) *roleBuilder {
	return &roleBuilder{
		typeBuilder: makeTypeBuilder(&role.typ),
		role:        role,
	}
}

func (rb *roleBuilder) Grant(kind GrantKind, objects []QName, fields []FieldName, comment ...string) IRoleBuilder {
	_ = newGrant(kind, objects, fields, rb.role, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(objects []QName, comment ...string) IRoleBuilder {
	gg := make(map[GrantKind]*grant)

	for _, o := range QNamesFrom(objects...) {
		t := rb.role.app.Type(o)
		if t == nil {
			panic(fmt.Errorf("%w: %v", ErrTypeNotFound, o))
		}

		if _, ok := t.(IStructure); ok { // or IRecord??
			for k := GrantKind_Insert; k <= GrantKind_Select; k++ {
				g := gg[k]
				if g == nil {
					g = newGrant(k, QNames{o}, nil, rb.role, comment...)
					gg[k] = g
				} else {
					g.objects = append(g.objects, o)
				}
			}
		}

		if _, ok := t.(IFunction); ok {
			g := gg[GrantKind_Execute]
			if g == nil {
				g = newGrant(GrantKind_Execute, QNames{o}, nil, rb.role, comment...)
				gg[GrantKind_Execute] = g
			} else {
				g.objects = append(g.objects, o)
			}
		}

		if _, ok := t.(IWorkspace); ok {
			for k := GrantKind_Insert; k <= GrantKind_Execute; k++ {
				g := gg[k]
				if g == nil {
					g = newGrant(k, QNames{o}, nil, rb.role, comment...)
					gg[k] = g
				} else {
					g.objects = append(g.objects, o)
				}
			}
		}
	}

	return rb
}

func (rb *roleBuilder) GrantRoles(roles []QName, comment ...string) IRoleBuilder {
	_ = newGrant(GrantKind_Role, roles, nil, rb.role, comment...)
	return rb
}
