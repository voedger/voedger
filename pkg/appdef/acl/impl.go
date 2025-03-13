/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// Returns is operation allowed on type for specified roles in specified workspace.
//
// Roles list should be expanded to include all inherited roles by caller.
//
// If operation is allowed, but with some fields restriction, then returns map of allowed fields, else returned fields map is nil.
func checkOperationOnTypeForRoles(ws appdef.IWorkspace, op appdef.OperationKind, t appdef.IType, roles appdef.QNames) (allowed bool, fields map[appdef.FieldName]bool) {

	if roles.Contains(appdef.QNameRoleSystem) {
		// nothing else matters
		return true, nil
	}

	var (
		resFields     appdef.IWithFields
		allowedFields map[appdef.FieldName]bool
	)
	if wf, ok := t.(appdef.IWithFields); ok {
		resFields = wf
		allowedFields = map[appdef.FieldName]bool{}
	}

	result := false

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

	if result && (resFields != nil) && (allowedFields != nil) && (len(allowedFields) == resFields.FieldCount()) {
		// all fields are allowed, should return nil
		allowedFields = nil
	}

	return result, allowedFields
}
