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

func newRole(app *appDef, ws *workspace, name QName) *role {
	r := &role{
		typ:      makeType(app, ws, name, TypeKind_Role),
		aclRules: make([]*aclRule, 0),
	}
	ws.appendType(r)
	return r
}

func (r role) ACL(cb func(IACLRule) bool) {
	for _, p := range r.aclRules {
		if !cb(p) {
			break
		}
	}
}

func (r *role) AncRoles() (roles []QName) {
	for _, p := range r.aclRules {
		if p.ops.Contains(OperationKind_Inherits) {
			for _, n := range p.Resources().On() {
				roles = append(roles, n)
			}
		}
	}
	return roles
}

func (r *role) appendACL(rule *aclRule) {
	r.aclRules = append(r.aclRules, rule)
	r.ws.appendACL(rule)
}

func (r *role) grant(ops []OperationKind, resources []QName, fields []FieldName, comment ...string) {
	r.appendACL(newGrant(ops, resources, fields, r, comment...))
}

func (r *role) grantAll(resources []QName, comment ...string) {
	r.appendACL(newGrantAll(resources, r, comment...))
}

func (r *role) revoke(ops []OperationKind, resources []QName, fields []FieldName, comment ...string) {
	r.appendACL(newRevoke(ops, resources, fields, r, comment...))
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

func (rb *roleBuilder) Revoke(ops []OperationKind, resources []QName, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.revoke(ops, resources, fields, comment...)
	return rb
}

func (rb *roleBuilder) RevokeAll(resources []QName, comment ...string) IRoleBuilder {
	rb.role.revokeAll(resources, comment...)
	return rb
}
