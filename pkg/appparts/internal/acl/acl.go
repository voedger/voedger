/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"fmt"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

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

// Returns true if specified operation is allowed in specified workspace on specified resource for any of specified roles.
//
// If resource is any structure and operation is UPDATE, INSERT or SELECT, then if fields list specified, then result consider it,
// else fields list is ignored.
//
// If some error in arguments, (ws or resource not found, operation is not applicable to resource, etc…) then error is returned.
func IsOperationAllowed(ws appdef.IWorkspace, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, rol []appdef.QName) (bool, error) {

	t := ws.Type(res)
	if t == appdef.NullType {
		return false, appdef.ErrNotFound("resource «%s» in %v", res, ws)
	}

	var str appdef.IStructure
	switch op {
	case appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select:
		if s, ok := t.(appdef.IStructure); ok {
			str = s
		} else {
			return false, appdef.ErrIncompatible("%v is not a structure", t)
		}
		for _, f := range fld {
			if str.Field(f) == nil {
				return false, appdef.ErrNotFound("field «%s» in %v", f, str)
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
		if str != nil {
			for _, f := range str.Fields() {
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
									if str != nil {
										if rule.Filter().HasFields() {
											// allow for specified fields only
											for _, f := range rule.Filter().Fields() {
												allowedFields[f] = true
											}
										} else {
											// allow for all fields
											for _, f := range str.Fields() {
												allowedFields[f.Name()] = true
											}
										}
									}
								case appdef.PolicyKind_Deny:
									if str != nil {
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
	if str != nil {
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
			for _, fld := range str.Fields() {
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

// here to avid memory consumption for returning []allowedField and []effectiveRole
func logVerboseDenyReason(op appdef.OperationKind, resource appdef.QName, allowed []appdef.FieldName, requestedFields []string, roles []appdef.QName, ws appdef.IWorkspace) {
	entity := resource.String()
	for _, reqField := range requestedFields {
		if !slices.Contains(allowed, reqField) {
			entity += "." + reqField
			break
		}
	}
	logger.Verbose(fmt.Sprintf("ws %s: %s on %s by %s -> deny", ws.Descriptor(), op, entity, roles))
}
