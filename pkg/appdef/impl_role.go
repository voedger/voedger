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

func (r role) ACLForResources(resources []QName, ops ...OperationKind) []IACLRule {
	pp := make([]IACLRule, 0)
	for _, p := range r.aclRules {
		if p.Resources().On().ContainsAny(resources...) && p.ops.ContainsAny(ops...) {
			pp = append(pp, p)
		}
	}
	return pp
}

func (r *role) appendACL(rule *aclRule) {
	r.aclRules = append(r.aclRules, rule)
	r.app.appendACL(rule)
}

func (r *role) grant(ops []OperationKind, resources []QName, fields []FieldName, comment ...string) {
	r.appendACL(newGrant(ops, resources, fields, r, comment...))
}

func (r *role) grantAll(resources []QName, comment ...string) {
	r.appendACL(newGrantAll(resources, r, comment...))
}

func (r *role) revoke(ops []OperationKind, resources []QName, comment ...string) {
	r.appendACL(newRevoke(ops, resources, nil, r, comment...))
}

func (r *role) revokeAll(resources []QName, comment ...string) {
	r.appendACL(newRevokeAll(resources, r, comment...))
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

func (rb *roleBuilder) Grant(ops []OperationKind, resources []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(ops, resources, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(resource []QName, comment ...string) IRoleBuilder {
	rb.role.grantAll(resource, comment...)
	return rb
}

func (rb *roleBuilder) Revoke(ops []OperationKind, resources []QName, comment ...string) IRoleBuilder {
	rb.role.revoke(ops, resources, comment...)
	return rb
}

func (rb *roleBuilder) RevokeAll(resources []QName, comment ...string) IRoleBuilder {
	rb.role.revokeAll(resources, comment...)
	return rb
}
