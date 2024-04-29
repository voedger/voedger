/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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

func (r role) PrivilegesOn(on []QName, kind ...PrivilegeKind) []IPrivilege {
	pp := make([]IPrivilege, 0)
	for _, p := range r.privileges {
		if p.On().ContainsAny(on...) && p.kinds.ContainsAny(kind...) {
			pp = append(pp, p)
		}
	}
	return pp
}

func (r *role) appendPrivilege(p *privilege) {
	r.privileges = append(r.privileges, p)
	r.app.appendPrivilege(p)
}

func (r *role) grant(kinds []PrivilegeKind, on []QName, fields []FieldName, comment ...string) {
	r.appendPrivilege(newGrant(kinds, on, fields, r, comment...))
}

func (r *role) grantAll(on []QName, comment ...string) {
	r.appendPrivilege(newGrantAll(on, r, comment...))
}

func (r *role) revoke(kinds []PrivilegeKind, on []QName, comment ...string) {
	r.appendPrivilege(newRevoke(kinds, on, nil, r, comment...))
}

func (r *role) revokeAll(on []QName, comment ...string) {
	r.appendPrivilege(newRevokeAll(on, r, comment...))
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

func (rb *roleBuilder) Revoke(kinds []PrivilegeKind, on []QName, comment ...string) IRoleBuilder {
	rb.role.revoke(kinds, on, comment...)
	return rb
}

func (rb *roleBuilder) RevokeAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.revokeAll(on, comment...)
	return rb
}
