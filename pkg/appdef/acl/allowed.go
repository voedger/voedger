/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
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

	allowedFields := map[appdef.FieldName]any{}

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

	result := false

	if slices.Contains(roles, appdef.QNameRoleSystem) {
		// nothing else matters
		result = true
		if resFields != nil {
			for _, f := range resFields.Fields() {
				allowedFields[f.Name()] = true
			}
		}
	} else {
		stack := map[appdef.QName]bool{}

		var acl func(ws appdef.IWorkspace)

		acl = func(ws appdef.IWorkspace) {
			if !stack[ws.QName()] {
				stack[ws.QName()] = true

				for _, anc := range ws.Ancestors() {
					acl(anc)
				}

				for _, rule := range ws.ACL() {
					if rule.Op(op) {
						if rule.Filter().Match(t) {
							if roles.Contains(rule.Principal().QName()) {
								switch rule.Policy() {
								case appdef.PolicyKind_Allow:
									result = true
									if resFields != nil {
										if rule.Filter().HasFields() {
											// allow for specified fields only
											for _, f := range rule.Filter().Fields() {
												allowedFields[f] = true
											}
										} else {
											// allow for all fields
											for _, f := range resFields.Fields() {
												allowedFields[f.Name()] = true
											}
										}
									}
								case appdef.PolicyKind_Deny:
									if resFields != nil {
										if rule.Filter().HasFields() {
											// partially deny, only specified fields
											for _, f := range rule.Filter().Fields() {
												delete(allowedFields, f)
											}
											result = len(allowedFields) > 0
										} else {
											// full deny, for all fields
											clear(allowedFields)
											result = false
										}
									} else {
										result = false
									}
								}
							}
						}
					}
				}
			}
		}
		acl(ws)
	}

	var allowed []appdef.FieldName
	if resFields != nil {
		if result {
			if len(fld) > 0 {
				for _, f := range fld {
					if _, ok := allowedFields[f]; !ok {
						result = false
						break
					}
				}
			}
		}
		if len(allowedFields) > 0 {
			allowed = make([]appdef.FieldName, 0, len(allowedFields))
			for _, fld := range resFields.Fields() {
				f := fld.Name()
				if _, ok := allowedFields[f]; ok {
					allowed = append(allowed, f)
				}
			}
		}
	}

	if !result && logger.IsVerbose() {
		logVerboseDenyReason(op, res, allowed, fld, roles, ws)
	}

	return result, nil
}
