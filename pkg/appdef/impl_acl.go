/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"iter"
	"slices"

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

func (f aclFilter) Fields() iter.Seq[FieldName] { return slices.Values(f.fields) }

func (f aclFilter) HasFields() bool { return len(f.fields) > 0 }

func (f aclFilter) String() string {
	if f.HasFields() {
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
	ws        *workspace
}

func newACLRule(ws *workspace, ops []OperationKind, policy PolicyKind, flt IFilter, fields []FieldName, principal *role, comment ...string) *aclRule {
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

	if flt == nil {
		panic(ErrMissed("filter"))
	}

	acl := &aclRule{
		comment:   makeComment(comment...),
		policy:    policy,
		ops:       opSet,
		flt:       newAclFilter(flt, fields),
		principal: principal,
		ws:        ws,
	}

	for t := range FilterMatches(acl.Filter(), ws.Types()) {
		if err := acl.validateOnType(t); err != nil {
			panic(err)
		}
	}

	return acl
}

func newGrant(ws *workspace, ops []OperationKind, flt IFilter, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ws, ops, PolicyKind_Allow, flt, fields, principal, comment...)
}

func newRevoke(ws *workspace, ops []OperationKind, flt IFilter, fields []FieldName, principal *role, comment ...string) *aclRule {
	return newACLRule(ws, ops, PolicyKind_Deny, flt, fields, principal, comment...)
}

func newACLRuleAll(ws *workspace, policy PolicyKind, flt IFilter, principal *role, comment ...string) *aclRule {
	if flt == nil {
		panic(ErrMissed("filter"))
	}

	t := FirstFilterMatch(flt, ws.LocalTypes())
	if t == nil {
		panic(ErrFilterHasNoMatches(flt, ws))
	}

	ops := allACLOperationsOnType(t)
	if ops.Len() == 0 {
		panic(ErrACLUnsupportedType(t))
	}

	return newACLRule(ws, ops.AsArray(), policy, flt, nil, principal, comment...)
}

func newGrantAll(ws *workspace, flt IFilter, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(ws, PolicyKind_Allow, flt, principal, comment...)
}

func newRevokeAll(ws *workspace, flt IFilter, principal *role, comment ...string) *aclRule {
	return newACLRuleAll(ws, PolicyKind_Deny, flt, principal, comment...)
}

func (acl aclRule) Filter() IACLFilter { return acl.flt }

func (acl aclRule) Op(op OperationKind) bool { return acl.ops.Contains(op) }

func (acl aclRule) Ops() iter.Seq[OperationKind] { return acl.ops.Values() }

func (acl aclRule) Policy() PolicyKind { return acl.policy }

func (acl aclRule) Principal() IRole { return acl.principal }

func (acl aclRule) String() string {
	// GRANT [Select] ON QNAMES(test.doc) TO test.reader
	// REVOKE [INSERT UPDATE SELECT] QNAMES(test.doc)([field1]) FROM test.writer
	s := fmt.Sprintf("%s %s ON %s", acl.Policy().ActionString(), acl.ops, acl.Filter())
	switch acl.policy {
	case PolicyKind_Deny:
		s += " FROM "
	default:
		s += " TO "
	}
	s += fmt.Sprint(acl.Principal().QName())
	return s
}

func (acl aclRule) Workspace() IWorkspace { return acl.ws }

// validates ACL rule.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not to be proceed with ACL. See validateOnType
func (acl aclRule) validate() (err error) {
	cnt := 0
	for t := range FilterMatches(acl.Filter(), acl.Workspace().Types()) {
		err = errors.Join(err, acl.validateOnType(t))
		cnt++
	}

	if (err == nil) && (cnt == 0) {
		return ErrFilterHasNoMatches(acl.Filter(), acl.Workspace())
	}

	return err
}

// validates ACL rule on the filtered type.
//
// # Error if:
//   - filtered type is not supported by ACL
//   - ACL operations are not compatible with the filtered type
//   - some specified field is not found in the filtered type
func (acl aclRule) validateOnType(t IType) error {
	allOps := allACLOperationsOnType(t)
	if allOps.Len() == 0 {
		return ErrACLUnsupportedType(t)
	}
	for op := range acl.Ops() {
		if !allOps.Contains(op) {
			return ErrIncompatible("operation %v and %v", op.TrimString(), t)
		}
	}
	if acl.Filter().HasFields() {
		if fields, ok := t.(IFields); ok {
			for f := range acl.Filter().Fields() {
				if fields.Field(f) == nil {
					return ErrNotFound("field «%v» in %v", f, t)
				}
			}
		}
	}
	return nil
}
