/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "slices"

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
