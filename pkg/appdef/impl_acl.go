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
//   - IACLRule
type aclRule struct {
	comment
	ops       set.Set[OperationKind]
	policy    PolicyKind
	on        QNames
	fields    []FieldName
	principal *role
}

func newACLRule(ops []OperationKind, policy PolicyKind, on []QName, fields []FieldName, principal *role, comment ...string) *aclRule {
	opSet := set.From(ops...)
	if opSet.Len() == 0 {
		panic(ErrMissed("operations"))
	}

	names, err := validateACLResourceNames(principal.app, on...)
	if err != nil {
		panic(err)
	}

	res := principal.app.Type(names[0])
	allOps := allACLOperationsOnType(res)
	if !allOps.ContainsAll(opSet.AsArray()...) {
		panic(ErrIncompatible("operations «%s» with %v", opSet, res))
	}

	if len(fields) > 0 {
		if !opSet.ContainsAny(OperationKind_Select, OperationKind_Update) {
			panic(ErrIncompatible("fields are not applicable for operations «%s»", opSet))
		}
		if err := validateFieldNamesByTypes(principal.app, on, fields); err != nil {
			panic(err)
		}
	}

	acl := &aclRule{
		comment:   makeComment(comment...),
		policy:    policy,
		ops:       opSet,
		on:        names,
		fields:    fields,
		principal: principal,
	}
	return acl
}

func newGrant(ops []OperationKind, on []QName, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ops, PolicyKind_Allow, on, fields, principal, comment...)
}

func newRevoke(ops []OperationKind, on []QName, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ops, PolicyKind_Deny, on, fields, principal, comment...)
}

func newACLRuleAll(policy PolicyKind, on []QName, principal *role, comment ...string) *aclRule {
	names, err := validateACLResourceNames(principal.app, on...)
	if err != nil {
		panic(err)
	}

	allOps := allACLOperationsOnType(principal.app.Type(names[0]))

	return newACLRule(allOps.AsArray(), policy, names, nil, principal, comment...)
}

func newGrantAll(on []QName, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(PolicyKind_Allow, on, principal, comment...)
}

func newRevokeAll(on []QName, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(PolicyKind_Deny, on, principal, comment...)
}

func (g aclRule) Fields() []FieldName { return g.fields }

func (g aclRule) Ops() []OperationKind { return g.ops.AsArray() }

func (g aclRule) On() QNames { return g.on }

func (g aclRule) Policy() PolicyKind { return g.policy }

func (g aclRule) Principal() IRole { return g.principal }

func (g aclRule) String() string {
	switch g.policy {
	case PolicyKind_Deny:
		return fmt.Sprintf("%s %v on %v from %v", g.policy.ActionString(), g.ops, g.on, g.principal)
	default:
		return fmt.Sprintf("%s %v on %v to %v", g.policy.ActionString(), g.ops, g.on, g.principal)
	}
}
