/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Makes RateScopes from specified scopes.
func RateScopesFrom(scopes ...RateScope) RateScopes {
	rs := make(RateScopes, 0, len(scopes))
	for _, s := range scopes {
		if (s > RateScope_null) && (s < RateScope_count) {
			if !slices.Contains(rs, s) {
				rs = append(rs, s)
			}
		} else {
			panic(ErrOutOfBounds("rate scope «%v»", s))
		}
	}
	return rs
}

// Returns is rate scopes contains the specified scope.
func (rs RateScopes) Contains(s RateScope) bool {
	return slices.Contains(rs, s)
}

// Renders an RateScopes in human-readable form, without "RateScopes_" prefix,
// suitable for debugging or error messages
func (rs RateScopes) String() string {
	var ss string
	for i, s := range rs {
		if i > 0 {
			ss = strings.Join([]string{ss, s.TrimString()}, " ")
		} else {
			ss = s.TrimString()
		}
	}
	return fmt.Sprintf("[%s]", ss)
}

// Renders an RateScope in human-readable form, without `RateScope_` prefix,
// suitable for debugging or error messages
func (rs RateScope) TrimString() string {
	const pref = "RateScope" + "_"
	return strings.TrimPrefix(rs.String(), pref)
}

// validates object names for rate limit
func validateLimitNames(tt IWithTypes, names QNames) (err error) {
	var any = QNamesFrom(
		QNameANY,
		QNameAnyStructure,
		QNameAnyRecord,
		QNameAnyGDoc,
		QNameAnyCDoc,
		QNameAnyWDoc,
		QNameAnySingleton,
		QNameAnyView,
		QNameAnyFunction,
		QNameAnyCommand,
		QNameAnyQuery,
	)
	if len(names) == 0 {
		return ErrMissed("limit objects names")
	}
	for _, n := range names {
		t := tt.TypeByName(n)
		if t == nil {
			err = errors.Join(err,
				ErrNotFound("type «%v»", n))
			continue
		}

		switch t.Kind() {
		case TypeKind_Any:
			if !any.Contains(n) {
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
