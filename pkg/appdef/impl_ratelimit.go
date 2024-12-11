/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"iter"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// Implements:
//   - IRate
type rate struct {
	typ
	count  RateCount
	period RatePeriod
	scopes set.Set[RateScope]
}

func newRate(app *appDef, ws *workspace, name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) *rate {
	r := &rate{
		typ:    makeType(app, ws, name, TypeKind_Rate),
		count:  count,
		period: period,
		scopes: set.From(scopes...),
	}
	if r.scopes.Len() == 0 {
		r.scopes.Set(DefaultRateScopes...)
	}
	r.typ.comment.setComment(comment...)
	ws.appendType(r)
	return r
}

func (r rate) Count() RateCount {
	return r.count
}

func (r rate) Period() RatePeriod {
	return r.period
}

func (r rate) Scopes() iter.Seq[RateScope] {
	return r.scopes.Values()
}

// Implements:
//   - ILimit
type limit struct {
	typ
	flt  IFilter
	rate IRate
}

func newLimit(app *appDef, ws *workspace, name QName, flt IFilter, rate QName, comment ...string) *limit {
	if rate == NullQName {
		panic(ErrMissed("rate name"))
	}
	if flt == nil {
		panic(ErrMissed("filter"))
	}
	l := &limit{
		typ:  makeType(app, ws, name, TypeKind_Limit),
		flt:  flt,
		rate: Rate(app.Type, rate),
	}
	if l.rate == nil {
		panic(ErrNotFound("rate «%v»", rate))
	}
	for t := range FilterMatches(l.On(), ws.Types()) {
		if err := l.validateOnType(t); err != nil {
			panic(err)
		}
	}
	l.typ.comment.setComment(comment...)
	ws.appendType(l)
	return l
}

func (l limit) On() IFilter {
	return l.flt
}

func (l limit) Rate() IRate {
	return l.rate
}

// Validates limit.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not to be limited. See validateOnType
func (l limit) Validate() (err error) {
	cnt := 0
	for t := range FilterMatches(l.On(), l.Workspace().Types()) {
		err = errors.Join(err, l.validateOnType(t))
		cnt++
	}

	if (err == nil) && (cnt == 0) {
		return ErrFilterHasNoMatches(l.On(), l.Workspace())
	}

	return err
}

func (l limit) validateOnType(t IType) error {
	switch t.Kind() {
	case TypeKind_Command, TypeKind_Query: //ok
	case TypeKind_GDoc, TypeKind_CDoc, TypeKind_WDoc,
		TypeKind_GRecord, TypeKind_CRecord, TypeKind_WRecord,
		TypeKind_ViewRecord: //ok
	default:
		return ErrUnsupported("%v can not to be limited", t)
	}
	return nil
}
