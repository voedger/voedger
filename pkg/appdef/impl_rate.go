/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strings"
)

// Implements:
//   - IRate
type rate struct {
	typ
	count  RateCount
	period RatePeriod
	scopes RateScopes
}

func newRate(app *appDef, name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) *rate {
	r := &rate{
		typ:    makeType(app, name, TypeKind_Rate),
		count:  count,
		period: period,
		scopes: RateScopesFrom(scopes...),
	}
	r.typ.comment.setComment(comment...)
	app.appendType(r)
	return r
}

func (r rate) Count() RateCount {
	return r.count
}

func (r rate) Period() RatePeriod {
	return r.period
}

func (r rate) Scopes() RateScopes {
	return r.scopes
}

func (r rate) String() string {
	return fmt.Sprintf("%v %d per %v per %v", r.typ, r.count, r.period, r.scopes)
}

// Renders an RateScope in human-readable form, without `RateScope_` prefix,
// suitable for debugging or error messages
func (rs RateScope) TrimString() string {
	const pref = "RateScope" + "_"
	return strings.TrimPrefix(rs.String(), pref)
}
