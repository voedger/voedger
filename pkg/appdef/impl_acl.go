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
	kinds  set.Set[OperationKind]
	policy PolicyKind
	on     QNames
	fields []FieldName
	role   *role
}

func newACLRule(kind []OperationKind, policy PolicyKind, on []QName, fields []FieldName, role *role, comment ...string) *aclRule {
	pk := set.From(kind...)
	if pk.Len() == 0 {
		panic(ErrMissed("operation kinds"))
	}

	names, err := validateACLResourceNames(role.app, on...)
	if err != nil {
		panic(err)
	}

	o := role.app.Type(names[0])
	allPk := allOperationsOnType(o)
	if !allPk.ContainsAll(pk.AsArray()...) {
		panic(ErrIncompatible("operations «%s» with %v", pk, o))
	}

	if len(fields) > 0 {
		if !pk.ContainsAny(OperationKind_Select, OperationKind_Update) {
			panic(ErrIncompatible("fields are not applicable for operations «%s»", pk))
		}
		if err := validateFieldNamesByTypes(role.app, on, fields); err != nil {
			panic(err)
		}
	}

	acl := &aclRule{
		comment: makeComment(comment...),
		policy:  policy,
		kinds:   pk,
		on:      names,
		fields:  fields,
		role:    role,
	}
	return acl
}

func newGrant(kinds []OperationKind, on []QName, fields []FieldName, role *role, comment ...string) *aclRule {
	return newACLRule(kinds, PolicyKind_Allow, on, fields, role, comment...)
}

func newRevoke(kinds []OperationKind, on []QName, fields []FieldName, role *role, comment ...string) *aclRule {
	return newACLRule(kinds, PolicyKind_Deny, on, fields, role, comment...)
}

func newPrivilegeAll(policy PolicyKind, on []QName, role *role, comment ...string) *aclRule {
	names, err := validateACLResourceNames(role.app, on...)
	if err != nil {
		panic(err)
	}

	pk := allOperationsOnType(role.app.Type(names[0]))

	return newACLRule(pk.AsArray(), policy, names, nil, role, comment...)
}

func newGrantAll(on []QName, role *role, comment ...string) *aclRule {
	return newPrivilegeAll(PolicyKind_Allow, on, role, comment...)
}

func newRevokeAll(on []QName, role *role, comment ...string) *aclRule {
	return newPrivilegeAll(PolicyKind_Deny, on, role, comment...)
}

func (g aclRule) Fields() []FieldName { return g.fields }

func (g aclRule) Policy() PolicyKind { return g.policy }

func (g aclRule) Ops() []OperationKind { return g.kinds.AsArray() }

func (g aclRule) On() QNames { return g.on }

func (g aclRule) To() IRole { return g.role }

func (g aclRule) String() string {
	switch g.policy {
	case PolicyKind_Deny:
		return fmt.Sprintf("%s %v on %v from %v", g.policy.ActionString(), g.kinds, g.on, g.role)
	default:
		return fmt.Sprintf("%s %v on %v to %v", g.policy.ActionString(), g.kinds, g.on, g.role)
	}
}
