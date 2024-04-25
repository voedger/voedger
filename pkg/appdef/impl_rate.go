/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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
	if len(r.scopes) == 0 {
		r.scopes = DefaultRateScopes
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
