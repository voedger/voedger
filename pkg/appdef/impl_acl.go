/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Supports:
//   - IACLFilter
type aclFilter struct {
	IFilter
	fields []FieldName
}

func newAclFilter(flt IFilter, fields []FieldName) *aclFilter {
	return &aclFilter{flt, fields}
}

func (f aclFilter) Fields() []FieldName { return f.fields }

func (f aclFilter) String() string {
	if len(f.fields) > 0 {
		return fmt.Sprintf("%v(%v)", f.IFilter, f.fields)
	}
	return fmt.Sprint(f.IFilter)
}

// # Supports:
//   - IACLRule
type aclRule struct {
	comment
	ops       set.Set[OperationKind]
	policy    PolicyKind
	flt       *aclFilter
	principal *role
}

func newACLRule(ops []OperationKind, policy PolicyKind, flt IFilter, fields []FieldName, principal *role, comment ...string) *aclRule {
	opSet := set.From(ops...)
	if compatible, err := isCompatibleOperations(opSet); !compatible {
		panic(err)
	}

	if opSet.Contains(OperationKind_Inherits) && (policy != PolicyKind_Allow) {
		panic(ErrUnsupported("%s %s", policy.ActionString(), OperationKind_Inherits.TrimString()))
	}

	if opSet.ContainsAny(OperationKind_Inherits, OperationKind_Execute) && (len(fields) > 0) {
		panic(ErrIncompatible("operations %v with fields", opSet))
	}

	acl := &aclRule{
		comment:   makeComment(comment...),
		policy:    policy,
		ops:       opSet,
		flt:       newAclFilter(flt, fields),
		principal: principal,
	}
	return acl
}

func newGrant(ops []OperationKind, flt IFilter, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ops, PolicyKind_Allow, flt, fields, principal, comment...)
}

func newRevoke(ops []OperationKind, flt IFilter, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ops, PolicyKind_Deny, flt, fields, principal, comment...)
}

func newACLRuleAll(policy PolicyKind, flt IFilter, principal *role, comment ...string) *aclRule {
	t := FirstFilterMatch(flt, principal.Workspace().LocalTypes())
	if t == nil {
		panic(ErrFilterHasNoMatches(flt, principal.Workspace()))
	}

	ops := allACLOperationsOnType(t)
	if ops.Len() == 0 {
		panic(ErrACLUnsupportedType(t))
	}

	return newACLRule(ops.AsArray(), policy, flt, nil, principal, comment...)
}

func newGrantAll(flt IFilter, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(PolicyKind_Allow, flt, principal, comment...)
}

func newRevokeAll(flt IFilter, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(PolicyKind_Deny, flt, principal, comment...)
}

func (acl aclRule) Filter() IACLFilter { return acl.flt }

func (acl aclRule) Ops() []OperationKind { return acl.ops.AsArray() }

func (acl aclRule) Policy() PolicyKind { return acl.policy }

func (acl aclRule) Principal() IRole { return acl.principal }

func (acl aclRule) String() string {
	s := fmt.Sprint(acl.Policy().ActionString(), acl.ops, " on ", acl.Filter())
	switch acl.policy {
	case PolicyKind_Deny:
		s += " from "
	default:
		s += " to "
	}
	s += fmt.Sprint(acl.Principal())
	return s
}

// validates ACL rule.
//
// # Error if:
//   - filter has no matches in the workspace
//   - filtered type is not supported by ACL
//   - ACL operations are not compatible with the filtered type
//   - some specified field is not found in the filtered type
func (acl aclRule) validate() error {
	cnt := 0
	for t := range FilterMatches(acl.Filter(), acl.Principal().Workspace().Types()) {
		o := allACLOperationsOnType(t)
		if o.Len() == 0 {
			return ErrACLUnsupportedType(t)
		}
		if !o.ContainsAll(acl.Ops()...) {
			return ErrIncompatible("operations %v and %v", acl.ops, t)
		}
		if ff := acl.Filter().Fields(); len(ff) > 0 {
			if fields, ok := t.(IFields); ok {
				for _, f := range ff {
					if fields.Field(f) == nil {
						return ErrNotFound("field «%v» in %v", f, t)
					}
				}
			}
		}
		cnt++
	}

	if cnt == 0 {
		return ErrFilterHasNoMatches(acl.Filter(), acl.Principal().Workspace())
	}

	return nil
}
