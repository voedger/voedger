/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Supports:
//   - appdef.IACLFilter
type filter struct {
	appdef.IFilter
	fields []appdef.FieldName
}

func newFilter(flt appdef.IFilter, fields []appdef.FieldName) *filter {
	return &filter{flt, fields}
}

func (f filter) Fields() iter.Seq[appdef.FieldName] { return slices.Values(f.fields) }

func (f filter) HasFields() bool { return len(f.fields) > 0 }

func (f filter) String() string {
	if f.HasFields() {
		return fmt.Sprintf("%v(%v)", f.IFilter, f.fields)
	}
	return fmt.Sprint(f.IFilter)
}

// # Supports:
//   - appdef.IACLRule
type Rule struct {
	comments.WithComments
	ops       set.Set[appdef.OperationKind]
	policy    appdef.PolicyKind
	flt       *filter
	principal appdef.IRole
	ws        appdef.IWorkspace
}

func NewRule(ws appdef.IWorkspace, ops []appdef.OperationKind, policy appdef.PolicyKind, flt appdef.IFilter, fields []appdef.FieldName, principal appdef.IRole, comment ...string) *Rule {
	if !appdef.ACLOperations.ContainsAll(ops...) {
		panic(appdef.ErrUnsupported("ACL operations %v", ops))
	}

	opSet := set.From(ops...)
	if compatible, err := appdef.IsCompatibleOperations(opSet); !compatible {
		panic(err)
	}

	if opSet.Contains(appdef.OperationKind_Inherits) && (policy != appdef.PolicyKind_Allow) {
		panic(appdef.ErrUnsupported("%s %s", policy.ActionString(), appdef.OperationKind_Inherits.TrimString()))
	}

	if opSet.ContainsAny(appdef.OperationKind_Inherits, appdef.OperationKind_Execute) && (len(fields) > 0) {
		panic(appdef.ErrIncompatible("operations %v with fields", opSet))
	}

	if flt == nil {
		panic(appdef.ErrMissed("filter"))
	}

	acl := &Rule{
		WithComments: comments.MakeWithComments(comment...),
		policy:       policy,
		ops:          opSet,
		flt:          newFilter(flt, fields),
		principal:    principal,
		ws:           ws,
	}

	for t := range appdef.FilterMatches(acl.Filter(), ws.Types()) {
		if err := acl.validateOnType(t); err != nil {
			panic(err)
		}
	}

	if role, ok := principal.(interface{ appendACL(appdef.IACLRule) }); ok {
		role.appendACL(acl) // propagate ACL to role, workspace and app
	}

	return acl
}

func NewGrant(ws appdef.IWorkspace, ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, principal appdef.IRole, comment ...string) *Rule {
	return NewRule(ws, ops, appdef.PolicyKind_Allow, flt, fields, principal, comment...)
}

func NewRevoke(ws appdef.IWorkspace, ops []appdef.OperationKind, flt appdef.IFilter, fields []appdef.FieldName, principal appdef.IRole, comment ...string) *Rule {
	return NewRule(ws, ops, appdef.PolicyKind_Deny, flt, fields, principal, comment...)
}

func NewRuleAll(ws appdef.IWorkspace, policy appdef.PolicyKind, flt appdef.IFilter, principal appdef.IRole, comment ...string) *Rule {
	if flt == nil {
		panic(appdef.ErrMissed("filter"))
	}

	t := appdef.FirstFilterMatch(flt, ws.LocalTypes())
	if t == nil {
		panic(appdef.ErrFilterHasNoMatches("ACL", flt, ws))
	}

	ops := appdef.ACLOperationsForType(t.Kind())
	if ops.Len() == 0 {
		panic(appdef.ErrACLUnsupportedType(t))
	}

	return NewRule(ws, ops.AsArray(), policy, flt, nil, principal, comment...)
}

func NewGrantAll(ws appdef.IWorkspace, flt appdef.IFilter, principal appdef.IRole, comment ...string) *Rule {
	return NewRuleAll(ws, appdef.PolicyKind_Allow, flt, principal, comment...)
}

func NewRevokeAll(ws appdef.IWorkspace, flt appdef.IFilter, principal appdef.IRole, comment ...string) *Rule {
	return NewRuleAll(ws, appdef.PolicyKind_Deny, flt, principal, comment...)
}

func (acl Rule) Filter() appdef.IACLFilter { return acl.flt }

func (acl Rule) Op(op appdef.OperationKind) bool { return acl.ops.Contains(op) }

func (acl Rule) Ops() iter.Seq[appdef.OperationKind] { return acl.ops.Values() }

func (acl Rule) Policy() appdef.PolicyKind { return acl.policy }

func (acl Rule) Principal() appdef.IRole { return acl.principal }

func (acl Rule) String() string {
	// GRANT [Select] ON QNAMES(test.doc) TO test.reader
	// REVOKE [INSERT UPDATE SELECT] QNAMES(test.doc)([field1]) FROM test.writer
	s := fmt.Sprintf("%s %s ON %s", acl.Policy().ActionString(), acl.ops, acl.Filter())
	switch acl.policy {
	case appdef.PolicyKind_Deny:
		s += " FROM "
	default:
		s += " TO "
	}
	s += fmt.Sprint(acl.Principal().QName())
	return s
}

// validates ACL rule.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not to be proceed with ACL. See validateOnType
func (acl Rule) Validate() (err error) {
	cnt := 0
	for t := range appdef.FilterMatches(acl.Filter(), acl.Workspace().Types()) {
		err = errors.Join(err, acl.validateOnType(t))
		cnt++
	}

	if cnt == 0 {
		err = errors.Join(err, appdef.ErrFilterHasNoMatches("ACL", acl.Filter(), acl.Workspace()))
	}

	return err
}

func (acl Rule) Workspace() appdef.IWorkspace { return acl.ws }

// validates ACL rule on the filtered type.
//
// # Error if:
//   - filtered type is not supported by ACL
//   - ACL operations are not compatible with the filtered type
//   - some specified field is not found in the filtered type
func (acl Rule) validateOnType(t appdef.IType) error {
	allOps := appdef.ACLOperationsForType(t.Kind())
	if allOps.Len() == 0 {
		return appdef.ErrACLUnsupportedType(t)
	}
	for op := range acl.Ops() {
		if !allOps.Contains(op) {
			return appdef.ErrIncompatible("operation %v and %v", op.TrimString(), t)
		}
	}
	if acl.Filter().HasFields() {
		if fields, ok := t.(appdef.IFields); ok {
			for f := range acl.Filter().Fields() {
				if fields.Field(f) == nil {
					return appdef.ErrNotFound("field «%v» in %v", f, t)
				}
			}
		}
	}
	return nil
}
