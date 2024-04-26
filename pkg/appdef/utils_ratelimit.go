/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
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
