/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "slices"

func (app *appDef) IsOperationAllowed(op OperationKind, res QName, fld []FieldName, prc []QName) (bool, []FieldName) {
	result := false

	var str IStructure
	if op == OperationKind_Update || op == OperationKind_Select {
		str = app.Structure(res)
	} else {
		str = nil
	}

	fields := map[FieldName]any{}

	roles := QNamesFrom(prc...)

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
									fields[f] = true
								}
							} else {
								// allow for all fields
								for _, f := range str.Fields() {
									fields[f.Name()] = true
								}
							}
						}
					case PolicyKind_Deny:
						if str != nil {
							if len(rule.Resources().Fields()) > 0 {
								// partially deny, only specified fields
								for _, f := range rule.Resources().Fields() {
									delete(fields, f)
								}
								result = len(fields) > 0
							} else {
								// full deny, for all fields
								clear(fields)
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
					if _, ok := fields[f]; !ok {
						result = false
						break
					}
				}
			}
		}
		if len(fields) > 0 {
			allowed := make([]FieldName, 0, len(fields))
			for _, fld := range str.Fields() {
				f := fld.Name()
				if _, ok := fields[f]; ok {
					allowed = append(allowed, f)
				}
			}
			return result, allowed
		}
	}

	return result, nil
}
