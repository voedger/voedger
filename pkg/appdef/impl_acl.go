/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Implements:
//   - IResourcePattern
type resourcePattern struct {
	on     QNames
	fields []FieldName
}

func newResourcePattern(on []QName, fields []FieldName) *resourcePattern {
	return &resourcePattern{on: on, fields: fields}
}

func (r resourcePattern) On() QNames { return r.on }

func (r resourcePattern) Fields() []FieldName { return r.fields }

func (r resourcePattern) String() string {
	if len(r.fields) > 0 {
		return fmt.Sprintf("%v(%v)", r.on, r.fields)
	}
	return fmt.Sprint(r.on)
}

// # Implements:
//   - IACLRule
type aclRule struct {
	comment
	ops       set.Set[OperationKind]
	policy    PolicyKind
	resources *resourcePattern
	principal *role
}

func newACLRule(ops []OperationKind, policy PolicyKind, resources []QName, fields []FieldName, principal *role, comment ...string) *aclRule {
	opSet := set.From(ops...)
	if opSet.Len() == 0 {
		panic(ErrMissed("operations"))
	}

	names, err := validateACLResourceNames(principal.app.Type, resources...)
	if err != nil {
		panic(err)
	}

	if opSet.Contains(OperationKind_Inherits) && (policy != PolicyKind_Allow) {
		panic(ErrUnsupported("«%s» for «%s»", policy.ActionString(), OperationKind_Inherits.TrimString()))
	}

	res := principal.app.Type(names[0])
	allOps := allACLOperationsOnType(res)
	if !allOps.ContainsAll(opSet.AsArray()...) {
		panic(ErrIncompatible("operations «%s» with %v", opSet, res))
	}

	if len(fields) > 0 {
		if !opSet.ContainsAny(
			OperationKind_Insert, // #2747
			OperationKind_Update, OperationKind_Select) {
			panic(ErrIncompatible("fields are not applicable for operations «%s»", opSet))
		}
		if err := validateFieldNamesByTypes(principal.app.Type, resources, fields); err != nil {
			panic(err)
		}
	}

	acl := &aclRule{
		comment:   makeComment(comment...),
		policy:    policy,
		ops:       opSet,
		resources: newResourcePattern(names, fields),
		principal: principal,
	}
	return acl
}

func newGrant(ops []OperationKind, resources []QName, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ops, PolicyKind_Allow, resources, fields, principal, comment...)
}

func newRevoke(ops []OperationKind, resources []QName, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ops, PolicyKind_Deny, resources, fields, principal, comment...)
}

func newACLRuleAll(policy PolicyKind, resources []QName, principal *role, comment ...string) *aclRule {
	names, err := validateACLResourceNames(principal.app.Type, resources...)
	if err != nil {
		panic(err)
	}

	allOps := allACLOperationsOnType(principal.app.Type(names[0]))

	return newACLRule(allOps.AsArray(), policy, names, nil, principal, comment...)
}

func newGrantAll(resources []QName, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(PolicyKind_Allow, resources, principal, comment...)
}

func newRevokeAll(resources []QName, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(PolicyKind_Deny, resources, principal, comment...)
}

func (g aclRule) Ops() []OperationKind { return g.ops.AsArray() }

func (g aclRule) Policy() PolicyKind { return g.policy }

func (g aclRule) Principal() IRole { return g.principal }

func (g aclRule) Resources() IResourcePattern { return g.resources }

func (g aclRule) String() string {
	switch g.policy {
	case PolicyKind_Deny:
		return fmt.Sprintf("%s %v on %v from %v", g.policy.ActionString(), g.ops, g.resources, g.principal)
	default:
		return fmt.Sprintf("%s %v on %v to %v", g.policy.ActionString(), g.ops, g.resources, g.principal)
	}
}
