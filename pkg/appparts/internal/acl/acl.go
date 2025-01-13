/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iauthnz"
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

// Returns true if specified operation is allowed on specified resource for any of specified roles.
//
// If resource is any structure and operation is UPDATE or SELECT, then:
//   - if fields list specified, then result consider it,
//   - full list of allowed fields also returned,
//
// else fields list is ignored and nil allowedFields is returned.
//
// If some error in arguments, (resource or role not found, operation is not applicable to resource, etc…) then error is returned.
func IsOperationAllowed(app appdef.IAppDef, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, rol []appdef.QName) (bool, []appdef.FieldName, error) {

	t := app.Type(res)
	if t == appdef.NullType {
		return false, nil, appdef.ErrNotFound("resource «%s»", res)
	}

	var str appdef.IStructure
	switch op {
	case appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select:
		if s, ok := t.(appdef.IStructure); ok {
			str = s
		} else {
			return false, nil, appdef.ErrNotFound("structure «%q»", res)
		}
		for _, f := range fld {
			if str.Field(f) == nil {
				return false, nil, appdef.ErrNotFound("field «%q» in %q", f, str)
			}
		}
	case appdef.OperationKind_Execute:
		if _, ok := t.(appdef.IFunction); !ok {
			return false, nil, appdef.ErrNotFound("function «%q»", res)
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
		role := appdef.Role(app.Type, r)
		if role == nil {
			return false, nil, appdef.ErrNotFound("role «%q»", r)
		}
		roles.Add(RecursiveRoleAncestors(role)...)
	}

	if slices.Contains(roles, iauthnz.QNameRoleSystem) {
		// nothung else matters
		return true, fld, nil
	}

	result := false
	for rule := range app.ACL() {
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

	if !result && logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("%s for %s: [%s] -> deny", op, res, rolesToString(rol)))
		for rule := range app.ACL() {
			ops := []string{}
			for op := range rule.Ops() {
				ops = append(ops, op.String())
			}
			logger.Verbose(fmt.Sprintf("%v %s for %s: %s", ops, rule.Filter(), rule.Principal(), rule.Policy()))
		}
	}

	return result, allowed, nil
}

func rolesToString(roles []appdef.QName) string {
	buf := bytes.NewBufferString("")
	for i, role := range roles {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(role.String())
	}
	return buf.String()
}
