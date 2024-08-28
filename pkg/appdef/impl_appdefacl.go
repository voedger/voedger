/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "slices"

func (app *appDef) IsOperationAllowed(op OperationKind, res QName, fld []FieldName, prc []QName) (bool, []FieldName, error) {

	var str IStructure
	switch op {
	case OperationKind_Insert:
		if app.Structure(res) == nil {
			return false, nil, ErrNotFound("structure «%q»", res)
		}
	case OperationKind_Update, OperationKind_Select:
		str = app.Structure(res)
		if str == nil {
			return false, nil, ErrNotFound("structure «%q»", res)
		}
		for _, f := range fld {
			if str.Field(f) == nil {
				return false, nil, ErrNotFound("field «%q» in %q", f, str)
			}
		}
	case OperationKind_Execute:
		if app.Function(res) == nil {
			return false, nil, ErrNotFound("function «%q»", res)
		}
	default:
		return false, nil, ErrUnsupported("operation %q", op)
	}

	allowedFields := map[FieldName]any{}

	roles := QNamesFrom(prc...)
	if len(roles) == 0 {
		return false, nil, ErrMissed("participants")
	}
	for _, r := range roles {
		if app.Role(r) == nil {
			return false, nil, ErrNotFound("role «%q»", r)
		}
	}

	result := false
	app.ACL(func(rule IACLRule) bool {
		if slices.Contains(rule.Ops(), op) {
			if rule.Resources().On().Contains(res) {
				if roles.Contains(rule.Principal().QName()) {
					switch rule.Policy() {
					case PolicyKind_Allow:
						result = true
						if str != nil {
							if len(rule.Resources().Fields()) > 0 {
								// allow for specified fields only
								for _, f := range rule.Resources().Fields() {
									allowedFields[f] = true
								}
							} else {
								// allow for all fields
								for _, f := range str.Fields() {
									allowedFields[f.Name()] = true
								}
							}
						}
					case PolicyKind_Deny:
						if str != nil {
							if len(rule.Resources().Fields()) > 0 {
								// partially deny, only specified fields
								for _, f := range rule.Resources().Fields() {
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
		return true
	})

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
			allowed := make([]FieldName, 0, len(allowedFields))
			for _, fld := range str.Fields() {
				f := fld.Name()
				if _, ok := allowedFields[f]; ok {
					allowed = append(allowed, f)
				}
			}
			return result, allowed, nil
		}
	}

	return result, nil, nil
}
