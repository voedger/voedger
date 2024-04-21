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
	privileges []*privilege
}

func newRole(app *appDef, name QName) *role {
	r := &role{
		typ:        makeType(app, name, TypeKind_Role),
		privileges: make([]*privilege, 0),
	}
	app.appendType(r)
	return r
}

func (r role) Privileges(cb func(IPrivilege)) {
	for _, g := range r.privileges {
		cb(g)
	}
}

func (r role) PrivilegesByKind(k PrivilegeKind, cb func(IPrivilege)) {
	for _, g := range r.privileges {
		if g.Kind() == k {
			cb(g)
		}
	}
}

func (r role) PrivilegesFor(name QName) []IPrivilege {
	gg := make([]IPrivilege, 0)
	for _, g := range r.privileges {
		if g.On().Contains(name) {
			gg = append(gg, g)
		}
	}
	return gg
}

func (r *role) grant(kind PrivilegeKind, on []QName, fields []FieldName, comment ...string) {
	r.privileges = append(r.privileges, newGrant(kind, on, fields, r, comment...))
}

func (r *role) grantAll(on []QName, comment ...string) {
	gg := make(map[PrivilegeKind]*privilege)

	for _, o := range QNamesFrom(on...) {
		t := r.app.Type(o)
		if t == nil {
			panic(fmt.Errorf("%w: %v", ErrTypeNotFound, o))
		}

		switch t.Kind() {
		case TypeKind_GRecord, TypeKind_GDoc,
			TypeKind_CRecord, TypeKind_CDoc,
			TypeKind_WRecord, TypeKind_WDoc,
			TypeKind_ORecord, TypeKind_ODoc,
			TypeKind_Object,
			TypeKind_ViewRecord:
			for k := PrivilegeKind_Insert; k <= PrivilegeKind_Select; k++ {
				if g, ok := gg[k]; ok {
					g.on.Add(o)
				} else {
					g := newGrant(k, []QName{o}, nil, r, comment...)
					r.privileges = append(r.privileges, g)
					gg[k] = g
				}
			}
		case TypeKind_Command, TypeKind_Query:
			if g, ok := gg[PrivilegeKind_Execute]; ok {
				g.on.Add(o)
			} else {
				g := newGrant(PrivilegeKind_Execute, []QName{o}, nil, r, comment...)
				r.privileges = append(r.privileges, g)
				gg[PrivilegeKind_Execute] = g
			}
		case TypeKind_Workspace:
			for k := PrivilegeKind_Insert; k <= PrivilegeKind_Execute; k++ {
				if g, ok := gg[k]; ok {
					g.on.Add(o)
				} else {
					g := newGrant(k, []QName{o}, nil, r, comment...)
					r.privileges = append(r.privileges, g)
					gg[k] = g
				}
			}
		case TypeKind_Role:
			if g, ok := gg[PrivilegeKind_Inherits]; ok {
				g.on.Add(o)
			} else {
				g := newGrant(PrivilegeKind_Inherits, []QName{o}, nil, r, comment...)
				r.privileges = append(r.privileges, g)
				gg[PrivilegeKind_Inherits] = g
			}
		default:
			panic(fmt.Errorf("can not grant privileges on: %w: %v", ErrInvalidTypeKind, o))
		}
	}
}

func (r *role) revoke(kind PrivilegeKind, on []QName, fields []FieldName, comment ...string) {
	r.privileges = append(r.privileges, newRevoke(kind, on, fields, r, comment...))
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

func (rb *roleBuilder) Grant(kind PrivilegeKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(kind, on, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.grantAll(on, comment...)
	return rb
}

func (rb *roleBuilder) Revoke(kind PrivilegeKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.revoke(kind, on, fields, comment...)
	return rb
}
