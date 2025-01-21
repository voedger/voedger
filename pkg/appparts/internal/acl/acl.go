/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
)

// Returns recursive list of role ancestors for specified role.
//
// If role has no ancestors, then result contains only specified role.
// Result is alphabetically sorted list of role names.
func RecursiveRoleAncestors(role appdef.IRole) (roles appdef.QNames) {
	roles.Add(role.QName())
	app := role.App()
	for r := range role.Ancestors() {
		roles.Add(RecursiveRoleAncestors(appdef.Role(app.Type, r))...)
	}
	return roles
}

// Returns true if specified operation is allowed in specified workspace on specified resource for any of specified roles.
//
// If resource is any structure and operation is UPDATE or SELECT, then:
//   - if fields list specified, then result consider it,
//   - full list of allowed fields also returned,
//
// else fields list is ignored and nil allowedFields is returned.
//
// If some error in arguments, (resource or role not found, operation is not applicable to resource, etc…) then error is returned.
func IsOperationAllowed(ws appdef.IWorkspace, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, rol []appdef.QName) (bool, []appdef.FieldName, error) {

	t := ws.Type(res)
	if t == appdef.NullType {
		return false, nil, appdef.ErrNotFound("resource «%s» in %v", res, ws)
	}

	var str appdef.IStructure
	switch op {
	case appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select:
		if s, ok := t.(appdef.IStructure); ok {
			str = s
		} else {
			return false, nil, appdef.ErrIncompatible("%v is not a structure", t)
		}
		for _, f := range fld {
			if str.Field(f) == nil {
				return false, nil, appdef.ErrNotFound("field «%s» in %v", f, str)
			}
		}
	case appdef.OperationKind_Activate, appdef.OperationKind_Deactivate:
		if rec, ok := t.(appdef.IRecord); ok {
			if f := rec.Field(appdef.SystemField_IsActive); f == nil {
				return false, nil, appdef.ErrNotFound("field «%s» in %v", appdef.SystemField_IsActive, rec)
			}
		} else {
			return false, nil, appdef.ErrIncompatible("%v is not a record", t)
		}
	case appdef.OperationKind_Execute:
		if _, ok := t.(appdef.IFunction); !ok {
			return false, nil, appdef.ErrIncompatible("%v is not a function", t)
		}
	default:
		return false, nil, appdef.ErrUnsupported("operation %q", op)
	}

	allowedFields := map[appdef.FieldName]any{}

	roles := appdef.QNamesFrom(rol...)

	if len(roles) == 0 {
		return false, nil, appdef.ErrMissed("participants")
	}
	for _, r := range roles {
		role := appdef.Role(ws.Type, r)
		if role != nil {
			roles.Add(RecursiveRoleAncestors(role)...)
		}
	}

	result := false

	if slices.Contains(roles, appdef.QNameRoleSystem) {
		// nothung else matters
		result = true
		if str != nil {
			for f := range str.Fields() {
				allowedFields[f.Name()] = true
			}
		}
	} else {
		stack := map[appdef.QName]bool{}

		var acl func(ws appdef.IWorkspace)

		acl = func(ws appdef.IWorkspace) {
			if !stack[ws.QName()] {
				stack[ws.QName()] = true

				for anc := range ws.Ancestors() {
					acl(anc)
				}

				for rule := range ws.ACL() {
					if rule.Op(op) {
						if rule.Filter().Match(t) {
							if roles.Contains(rule.Principal().QName()) {
								switch rule.Policy() {
								case appdef.PolicyKind_Allow:
									result = true
									if str != nil {
										if rule.Filter().HasFields() {
											// allow for specified fields only
											for f := range rule.Filter().Fields() {
												allowedFields[f] = true
											}
										} else {
											// allow for all fields
											for f := range str.Fields() {
												allowedFields[f.Name()] = true
											}
										}
									}
								case appdef.PolicyKind_Deny:
									if str != nil {
										if rule.Filter().HasFields() {
											// partially deny, only specified fields
											for f := range rule.Filter().Fields() {
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
			for fld := range str.Fields() {
				f := fld.Name()
				if _, ok := allowedFields[f]; ok {
					allowed = append(allowed, f)
				}
			}
		}
	}

	// nnv: logging should be moved to caller
	// if !result && logger.IsVerbose() {
	// 	logger.Verbose(fmt.Sprintf("%s for %s: [%s] -> deny", op, res, rolesToString(rol)))
	// 	for rule := range ws.App().ACL() {
	// 		logger.Verbose(fmt.Sprintf("%v : %v", rule.Workspace(), rule))
	// 	}
	// }

	return result, allowed, nil
}
