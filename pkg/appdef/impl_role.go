/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "errors"

// # Supports:
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
	for _, acl := range r.aclRules {
		if acl.ops.Contains(OperationKind_Inherits) {
			for role := range Roles(FilterMatches(acl.Filter(), r.Workspace().Types())) {
				roles = append(roles, role.QName())
			}
		}
	}
	return roles
}

func (r *role) appendACL(rule *aclRule) {
	r.aclRules = append(r.aclRules, rule)
	r.ws.appendACL(rule)
}

func (r *role) grant(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) {
	r.appendACL(newGrant(r.ws, ops, flt, fields, r, comment...))
}

func (r *role) grantAll(flt IFilter, comment ...string) {
	r.appendACL(newGrantAll(r.ws, flt, r, comment...))
}

func (r *role) revoke(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) {
	r.appendACL(newRevoke(r.ws, ops, flt, fields, r, comment...))
}

func (r *role) revokeAll(flt IFilter, comment ...string) {
	r.appendACL(newRevokeAll(r.ws, flt, r, comment...))
}

// validates role.
//
// # Error if:
//   - ACL rule is not valid
func (r role) Validate() (err error) {
	for _, p := range r.aclRules {
		if e := p.validate(); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

// # Supports:
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

func (rb *roleBuilder) Grant(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.grant(ops, flt, fields, comment...)
	return rb
}

func (rb *roleBuilder) GrantAll(flt IFilter, comment ...string) IRoleBuilder {
	rb.role.grantAll(flt, comment...)
	return rb
}

func (rb *roleBuilder) Revoke(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) IRoleBuilder {
	rb.role.revoke(ops, flt, fields, comment...)
	return rb
}

func (rb *roleBuilder) RevokeAll(flt IFilter, comment ...string) IRoleBuilder {
	rb.role.revokeAll(flt, comment...)
	return rb
}
