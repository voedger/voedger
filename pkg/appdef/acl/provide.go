/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"iter"
	"maps"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

// Returns true if specified operation is allowed in specified workspace on specified resource for any of specified roles.
//
// If resource is any structure and operation is UPDATE, INSERT or SELECT, then if fields list specified, then result consider it,
// else fields list is ignored.
//
// If some error in arguments, (resource not found, operation is not applicable to resource, etc…) then error is returned.
func IsOperationAllowed(ws appdef.IWorkspace, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, rol []appdef.QName) (bool, error) {

	t := ws.Type(res)
	if t == appdef.NullType {
		return false, appdef.ErrNotFound("resource «%s» in %v", res, ws)
	}

	var resFields appdef.IWithFields
	switch op {
	case appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select:
		if wf, ok := t.(appdef.IWithFields); ok {
			resFields = wf
		} else {
			return false, appdef.ErrIncompatible("%v has no fields", t)
		}
		for _, f := range fld {
			if resFields.Field(f) == nil {
				return false, appdef.ErrNotFound("field «%s» in %v", f, t)
			}
		}
	case appdef.OperationKind_Activate, appdef.OperationKind_Deactivate:
		// #3148: appparts: ACTIVATE/DEACTIVATE in IsOperationAllowed
		if rec, ok := t.(appdef.IRecord); ok {
			if f := rec.Field(appdef.SystemField_IsActive); f == nil {
				return false, appdef.ErrNotFound("field «%s» in %v", appdef.SystemField_IsActive, rec)
			}
		} else {
			return false, appdef.ErrIncompatible("%v is not a record", t)
		}
	case appdef.OperationKind_Execute:
		if _, ok := t.(appdef.IFunction); !ok {
			return false, appdef.ErrIncompatible("%v is not a function", t)
		}
	default:
		return false, appdef.ErrUnsupported("operation %q", op)
	}

	roles := appdef.QNamesFrom(rol...)

	if len(roles) == 0 {
		return false, appdef.ErrMissed("participants")
	}
	for _, r := range roles {
		role := appdef.Role(ws.Type, r)
		if role != nil {
			roles.Add(RecursiveRoleAncestors(role, ws)...)
		}
	}

	result, allowedFields := checkOperationOnTypeForRoles(ws, op, t, roles)
	failedField := ""

	if result && len(allowedFields) > 0 {
		for _, f := range fld {
			if !allowedFields[f] {
				result = false
				failedField = f
				break
			}
		}
	}

	if !result && logger.IsVerbose() {
		logVerboseDenyReason(op, res, failedField, roles, ws)
	}

	return result, nil
}

// [~server.apiv2.role/cmp.publishedTypes~impl]
// PublishedTypes lists the resources allowed to the published role in the workspace and ancestors (including resources available to non-authenticated requests).
//
// Types enumerated in alphabetical order.
//
// When returned fields is nil, it means all fields are allowed.
//
// Resources are:
//   - Documents and records
//   - Objects (function arguments)
//   - Views
//   - Commands and queries
func PublishedTypes(ws appdef.IWorkspace, role appdef.QName) iter.Seq2[appdef.IType,
	iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]] {

	roles := appdef.QNamesFrom(role)
	if r := appdef.Role(ws.Type, role); r != nil {
		roles = RecursiveRoleAncestors(r, ws)
	}

	types := map[appdef.IType]map[appdef.OperationKind]*[]appdef.FieldName{}

	for _, t := range ws.Types() {
		if k := t.Kind(); publushedTypes.Contains(k) {
			for o := range appdef.ACLOperationsForType(k).Values() {
				if ok, fields := checkOperationOnTypeForRoles(ws, o, t, roles); ok {
					if _, found := types[t]; !found {
						types[t] = map[appdef.OperationKind]*[]appdef.FieldName{}
					}
					if fields == nil {
						types[t][o] = nil
					} else {
						fld := make([]appdef.FieldName, 0, len(fields))
						for _, f := range t.(appdef.IWithFields).Fields() {
							if fn := f.Name(); fields[fn] {
								fld = append(fld, fn)
							}
						}
						types[t][o] = &fld
					}
				}
			}
		}
	}

	return func(visitType func(appdef.IType, iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]) bool) {
		ordered := slices.Collect(maps.Keys(types))
		slices.SortFunc(ordered, func(t1, t2 appdef.IType) int { return appdef.CompareQName(t1.QName(), t2.QName()) })
		for _, t := range ordered {
			if ops, ok := types[t]; ok && (len(ops) > 0) {
				if !visitType(t,
					func(visitOp func(appdef.OperationKind, *[]appdef.FieldName) bool) {
						for o := range appdef.ACLOperationsForType(t.Kind()).Values() {
							if fields, ok := ops[o]; ok {
								if !visitOp(o, fields) {
									return
								}
							}
						}
					}) {
					return
				}
			}
		}
	}
}

// Returns recursive list of role ancestors for specified role in the specified workspace.
//
// Role inheritance provided by `GRANT <role> TO <role>` statement.
// These inheritances statements can be provided in the specified workspace or in any of its ancestors.
//
// If role has no ancestors, then result contains only specified role.
// Result is alphabetically sorted list of role names.
func RecursiveRoleAncestors(role appdef.IRole, ws appdef.IWorkspace) (roles appdef.QNames) {
	roles.Add(role.QName())

	for _, acl := range ws.ACL() {
		if acl.Op(appdef.OperationKind_Inherits) && (acl.Principal() == role) {
			for _, t := range appdef.FilterMatches(acl.Filter(), ws.Types()) {
				if r, ok := t.(appdef.IRole); ok {
					roles.Add(RecursiveRoleAncestors(r, ws)...)
				}
			}
		}
	}

	for _, w := range ws.Ancestors() {
		roles.Add(RecursiveRoleAncestors(role, w)...)
	}

	return roles
}
