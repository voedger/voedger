/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"iter"
	"slices"
)

// # Supports:
//   - IRole
type role struct {
	typ
	acl []*aclRule
}

func newRole(app *appDef, ws *workspace, name QName) *role {
	r := &role{
		typ: makeType(app, ws, name, TypeKind_Role),
		acl: make([]*aclRule, 0),
	}
	ws.appendType(r)
	return r
}

func (r role) ACL() iter.Seq[IACLRule] {
	return func(yield func(IACLRule) bool) {
		for _, acl := range r.acl {
			if !yield(acl) {
				return
			}
		}
	}
}

func (r *role) Ancestors() iter.Seq[QName] {
	roles := QNames{}
	for _, acl := range r.acl {
		if acl.ops.Contains(OperationKind_Inherits) {
			switch acl.Filter().Kind() {
			case FilterKind_QNames:
				for q := range acl.Filter().QNames() {
					roles.Add(q)
				}
			default:
				// complex filter
				for role := range Roles(FilterMatches(acl.Filter(), r.Workspace().Types())) {
					roles.Add(role.QName())
				}
			}
		}
	}
	return slices.Values(roles)
}

func (r *role) appendACL(rule *aclRule) {
	r.acl = append(r.acl, rule)
}

func (r *role) grant(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) {
	acl := newGrant(r.ws, ops, flt, fields, r, comment...)
	r.appendACL(acl)
	r.ws.appendACL(acl)
}

func (r *role) grantAll(flt IFilter, comment ...string) {
	acl := newGrantAll(r.ws, flt, r, comment...)
	r.appendACL(acl)
	r.ws.appendACL(acl)
}

func (r *role) revoke(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) {
	acl := newRevoke(r.ws, ops, flt, fields, r, comment...)
	r.appendACL(acl)
	r.ws.appendACL(acl)
}

func (r *role) revokeAll(flt IFilter, comment ...string) {
	acl := newRevokeAll(r.ws, flt, r, comment...)
	r.appendACL(acl)
	r.ws.appendACL(acl)
}

// validates role.
//
// # Error if:
//   - ACL rule is not valid
func (r role) Validate() (err error) {
	for _, p := range r.acl {
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
