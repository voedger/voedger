/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"errors"
	"fmt"

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

func (f filter) Fields() []appdef.FieldName { return f.fields }

func (f filter) HasFields() bool { return len(f.fields) > 0 }

func (f filter) String() string {
	if f.HasFields() {
		return fmt.Sprintf("%v%v", f.IFilter, f.fields)
	}
	return fmt.Sprint(f.IFilter)
}

// # Supports:
//   - appdef.IACLRule
type Rule struct {
	comments.WithComments
	ops       []appdef.OperationKind
	opSet     set.Set[appdef.OperationKind]
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

	r := &Rule{
		WithComments: comments.MakeWithComments(comment...),
		policy:       policy,
		ops:          opSet.AsArray(),
		opSet:        opSet,
		flt:          newFilter(flt, fields),
		principal:    principal,
		ws:           ws,
	}

	for _, t := range appdef.FilterMatches(r.Filter(), ws.Types()) {
		if err := r.validateOnType(t); err != nil {
			panic(err)
		}
	}

	// propagate ACL to role, workspace and app
	type I interface{ AppendACL(appdef.IACLRule) }
	ws.(I).AppendACL(r)
	ws.App().(I).AppendACL(r)

	return r
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

	t := appdef.FirstFilterMatch(flt, ws.Types())
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

func (r Rule) Filter() appdef.IACLFilter { return r.flt }

func (r Rule) Op(op appdef.OperationKind) bool { return r.opSet.Contains(op) }

func (r Rule) Ops() []appdef.OperationKind { return r.ops }

func (r Rule) Policy() appdef.PolicyKind { return r.policy }

func (r Rule) Principal() appdef.IRole { return r.principal }

func (r Rule) String() string {
	// GRANT [Select] ON QNAMES(test.doc) TO test.reader
	// REVOKE [INSERT UPDATE SELECT] QNAMES(test.doc)([field1]) FROM test.writer
	s := fmt.Sprintf("%s %s ON %s", r.Policy().ActionString(), r.opSet, r.Filter())
	switch r.policy {
	case appdef.PolicyKind_Deny:
		s += " FROM "
	default:
		s += " TO "
	}
	s += fmt.Sprint(r.Principal().QName())
	return s
}

// validates ACL rule.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not to be proceed with ACL. See validateOnType
func (r Rule) Validate() (err error) {
	cnt := 0
	for _, t := range appdef.FilterMatches(r.Filter(), r.Workspace().Types()) {
		err = errors.Join(err, r.validateOnType(t))
		cnt++
	}

	if cnt == 0 {
		err = errors.Join(err, appdef.ErrFilterHasNoMatches("ACL", r.Filter(), r.Workspace()))
	}

	return err
}

func (r Rule) Workspace() appdef.IWorkspace { return r.ws }

// validates ACL rule on the filtered type.
//
// # Error if:
//   - filtered type is not supported by ACL
//   - ACL operations are not compatible with the filtered type
//   - some specified field is not found in the filtered type
func (r Rule) validateOnType(t appdef.IType) error {
	allOps := appdef.ACLOperationsForType(t.Kind())
	if allOps.Len() == 0 {
		return appdef.ErrACLUnsupportedType(t)
	}
	for _, op := range r.Ops() {
		if !allOps.Contains(op) {
			return appdef.ErrIncompatible("operation %v and %v", op.TrimString(), t)
		}
	}
	if r.Filter().HasFields() {
		if fields, ok := t.(appdef.IWithFields); ok {
			for _, f := range r.Filter().Fields() {
				if fields.Field(f) == nil {
					return appdef.ErrNotFound("field «%v» in %v", f, t)
				}
			}
		}
	}
	return nil
}
