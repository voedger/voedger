/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "github.com/voedger/voedger/pkg/goutils/set"

// Implements:
//   - IRate
type rate struct {
	typ
	count  RateCount
	period RatePeriod
	scopes set.Set[RateScope]
}

func newRate(app *appDef, name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) *rate {
	r := &rate{
		typ:    makeType(app, name, TypeKind_Rate),
		count:  count,
		period: period,
		scopes: set.From(scopes...),
	}
	if r.scopes.Len() == 0 {
		r.scopes.Set(DefaultRateScopes...)
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

func (r rate) Scopes() []RateScope {
	return r.scopes.AsArray()
}

// Implements:
//   - ILimit
type limit struct {
	typ
	on   QNames
	rate IRate
}

func newLimit(app *appDef, name QName, on []QName, rate QName, comment ...string) *limit {
	if rate == NullQName {
		panic(ErrMissed("rate name"))
	}
	l := &limit{
		typ:  makeType(app, name, TypeKind_Limit),
		on:   on,
		rate: app.Rate(rate),
	}
	if len(l.on) == 0 {
		panic(ErrMissed("limit objects names"))
	}
	if l.rate == nil {
		panic(ErrNotFound("rate «%v»", rate))
	}
	l.typ.comment.setComment(comment...)
	app.appendType(l)
	return l
}

func (l limit) On() QNames {
	return l.on
}

func (l limit) Rate() IRate {
	return l.rate
}

func (l limit) Validate() (err error) {
	return validateLimitNames(l.app, l.on)
}
