/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"strings"
)

// Renders an RateScope in human-readable form, without `RateScope_` prefix,
// suitable for debugging or error messages
func (rs RateScope) TrimString() string {
	const pref = "RateScope" + "_"
	return strings.TrimPrefix(rs.String(), pref)
}

// validates object names for rate limit
func validateLimitNames(ft FindType, names QNames) (err error) {
	var validAny = QNamesFrom(
		QNameANY,
		QNameAnyStructure, QNameAnyRecord,
		QNameAnyGDoc, QNameAnyCDoc, QNameAnyWDoc, QNameAnySingleton,
		QNameAnyView,
		QNameAnyFunction, QNameAnyCommand, QNameAnyQuery,
	)
	if len(names) == 0 {
		return ErrMissed("limit objects names")
	}
	for _, n := range names {
		t := ft(n)
		switch t.Kind() {
		case TypeKind_null:
			err = errors.Join(err,
				ErrNotFound("type «%v»", n))
		case TypeKind_Any:
			if !validAny.Contains(n) {
				err = errors.Join(err,
					ErrIncompatible("limit any «%v»", n))
			}
		case TypeKind_Command, TypeKind_Query: //ok
		case TypeKind_GDoc, TypeKind_CDoc, TypeKind_WDoc,
			TypeKind_GRecord, TypeKind_CRecord, TypeKind_WRecord,
			TypeKind_ViewRecord: //ok
		default:
			err = errors.Join(err,
				ErrIncompatible("limit «%v»", n))
		}
	}
	return err
}
