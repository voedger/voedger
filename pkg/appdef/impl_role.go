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

func (r role) ACL(cb func(IACLRule)) {
	for _, p := range r.aclRules {
		cb(p)
	}
}

func (r role) ACLForResources(on []QName, ops ...OperationKind) []IACLRule {
	pp := make([]IACLRule, 0)
	for _, p := range r.aclRules {
		if p.Resources().On().ContainsAny(on...) && p.ops.ContainsAny(ops...) {
			pp = append(pp, p)
		}
	}
	return pp
}

func (r *role) appendACL(rule *aclRule) {
	r.aclRules = append(r.aclRules, rule)
	r.app.appendACL(rule)
}

func (r *role) grant(ops []OperationKind, on []QName, fields []FieldName, comment ...string) {
	r.appendACL(newGrant(ops, on, fields, r, comment...))
}

func (r *role) grantAll(on []QName, comment ...string) {
	r.appendACL(newGrantAll(on, r, comment...))
}

func (r *role) revoke(ops []OperationKind, on []QName, comment ...string) {
	r.appendACL(newRevoke(ops, on, nil, r, comment...))
}

func (r *role) revokeAll(on []QName, comment ...string) {
	r.appendACL(newRevokeAll(on, r, comment...))
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

func (rb *roleBuilder) Grant(ops []OperationKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(ops, on, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.grantAll(on, comment...)
	return rb
}

func (rb *roleBuilder) Revoke(ops []OperationKind, on []QName, comment ...string) IRoleBuilder {
	rb.role.revoke(ops, on, comment...)
	return rb
}

func (rb *roleBuilder) RevokeAll(on []QName, comment ...string) IRoleBuilder {
	rb.role.revokeAll(on, comment...)
	return rb
}
