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
	grants []*privilege
}

func newRole(app *appDef, name QName) *role {
	r := &role{
		typ:    makeType(app, name, TypeKind_Role),
		grants: make([]*privilege, 0),
	}
	app.appendType(r)
	return r
}

func (r role) Privileges(cb func(IPrivilege)) {
	for _, g := range r.grants {
		cb(g)
	}
}

func (r role) PrivilegesByKind(k PrivilegeKind, cb func(IPrivilege)) {
	for _, g := range r.grants {
		if g.Kind() == k {
			cb(g)
		}
	}
}

func (r role) PrivilegesFor(name QName) []IPrivilege {
	gg := make([]IPrivilege, 0)
	for _, g := range r.grants {
		if g.Objects().Contains(name) {
			gg = append(gg, g)
		}
	}
	return gg
}

func (r *role) grant(kind PrivilegeKind, objects []QName, fields []FieldName, comment ...string) {
	r.grants = append(r.grants, newGrant(kind, objects, fields, r, comment...))
}

func (r *role) grantAll(objects []QName, comment ...string) {
	gg := make(map[PrivilegeKind]*privilege)

	for _, o := range QNamesFrom(objects...) {
		t := r.app.Type(o)
		if t == nil {
			panic(fmt.Errorf("%w: %v", ErrTypeNotFound, o))
		}

		if _, ok := t.(IStructure); ok { // or IRecord??
			for k := PrivilegeKind_Insert; k <= PrivilegeKind_Select; k++ {
				if g, ok := gg[k]; ok {
					g.objects.Add(o)
				} else {
					g := newGrant(k, []QName{o}, nil, r, comment...)
					r.grants = append(r.grants, g)
					gg[k] = g
				}
			}
		}

		if _, ok := t.(IFunction); ok {
			if g, ok := gg[PrivilegeKind_Execute]; ok {
				g.objects.Add(o)
			} else {
				g := newGrant(PrivilegeKind_Execute, []QName{o}, nil, r, comment...)
				r.grants = append(r.grants, g)
				gg[PrivilegeKind_Execute] = g
			}
		}

		if _, ok := t.(IWorkspace); ok {
			for k := PrivilegeKind_Insert; k <= PrivilegeKind_Execute; k++ {
				if g, ok := gg[k]; ok {
					g.objects.Add(o)
				} else {
					g := newGrant(k, []QName{o}, nil, r, comment...)
					r.grants = append(r.grants, g)
					gg[k] = g
				}
			}
		}
	}
}

func (r *role) grantRoles(roles []QName, comment ...string) {
	r.grants = append(r.grants, newGrant(PrivilegeKind_Role, roles, nil, r, comment...))
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

func (rb *roleBuilder) Grant(kind PrivilegeKind, objects []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(kind, objects, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(objects []QName, comment ...string) IRoleBuilder {
	rb.role.grantAll(objects, comment...)
	return rb
}

func (rb *roleBuilder) GrantRoles(roles []QName, comment ...string) IRoleBuilder {
	rb.role.grantRoles(roles, comment...)
	return rb
}
