/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IRole
type role struct {
	typ
	aclRules []*aclRule
}

func newRole(app *appDef, name QName) *role {
	r := &role{
		typ:      makeType(app, name, TypeKind_Role),
		aclRules: make([]*aclRule, 0),
	}
	app.appendType(r)
	return r
}

func (r role) Privileges(cb func(IACLRule)) {
	for _, p := range r.aclRules {
		cb(p)
	}
}

func (r role) PrivilegesOn(on []QName, kind ...OperationKind) []IACLRule {
	pp := make([]IACLRule, 0)
	for _, p := range r.aclRules {
		if p.On().ContainsAny(on...) && p.kinds.ContainsAny(kind...) {
			pp = append(pp, p)
		}
	}
	return pp
}

func (r *role) appendPrivilege(p *aclRule) {
	r.aclRules = append(r.aclRules, p)
	r.app.appendPrivilege(p)
}

func (r *role) grant(kinds []OperationKind, on []QName, fields []FieldName, comment ...string) {
	r.appendPrivilege(newGrant(kinds, on, fields, r, comment...))
}

func (r *role) grantAll(on []QName, comment ...string) {
	r.appendPrivilege(newGrantAll(on, r, comment...))
}

func (r *role) revoke(kinds []OperationKind, on []QName, comment ...string) {
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

func (rb *roleBuilder) Grant(kinds []OperationKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(kinds, on, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.grantAll(on, comment...)
	return rb
}

func (rb *roleBuilder) Revoke(kinds []OperationKind, on []QName, comment ...string) IRoleBuilder {
	rb.role.revoke(kinds, on, comment...)
	return rb
}

func (rb *roleBuilder) RevokeAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.revokeAll(on, comment...)
	return rb
}
