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
	for _, p := range r.privileges {
		cb(p)
	}
}

func (r role) PrivilegesOn(on QName, kind ...PrivilegeKind) []IPrivilege {
	pp := make([]IPrivilege, 0)
	for _, p := range r.privileges {
		if p.On().Contains(on) && p.Kinds().ContainsAny(kind...) {
			pp = append(pp, p)
		}
	}
	return pp
}

func (r *role) grant(kinds []PrivilegeKind, on []QName, fields []FieldName, comment ...string) {
	r.privileges = append(r.privileges, newGrant(kinds, on, fields, r, comment...))
}

func (r *role) grantAll(on []QName, comment ...string) {
	names := QNamesFrom(on...)
	if len(names) == 0 {
		panic(ErrPrivilegeOnMissed)
	}

	pk := PrivilegeKinds{}

	o := names[0]
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
		pk = PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}
	case TypeKind_Command, TypeKind_Query:
		pk = PrivilegeKinds{PrivilegeKind_Execute}
	case TypeKind_Workspace:
		pk = PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute}
	case TypeKind_Role:
		pk = PrivilegeKinds{PrivilegeKind_Inherits}
	default:
		panic(fmt.Errorf("can not grant privileges on: %w: %v", ErrInvalidTypeKind, o))
	}

	r.privileges = append(r.privileges, newGrant(pk, names, nil, r, comment...))
}

func (r *role) revoke(kinds []PrivilegeKind, on []QName, fields []FieldName, comment ...string) {
	r.privileges = append(r.privileges, newRevoke(kinds, on, fields, r, comment...))
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

func (rb *roleBuilder) Grant(kinds []PrivilegeKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(kinds, on, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.grantAll(on, comment...)
	return rb
}

func (rb *roleBuilder) Revoke(kinds []PrivilegeKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.revoke(kinds, on, fields, comment...)
	return rb
}
