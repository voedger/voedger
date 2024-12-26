/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Supports:
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

func (r rate) Scope(s RateScope) bool {
	return r.scopes.Contains(s)
}

func (r rate) Scopes() iter.Seq[RateScope] {
	return r.scopes.Values()
}

// # Supports:
//   - ILimitFilter
type limitFilter struct {
	IFilter
	opt LimitFilterOption
}

func newLimitFilter(opt LimitFilterOption, flt IFilter) *limitFilter {
	return &limitFilter{flt, opt}
}

func (f limitFilter) Option() LimitFilterOption { return f.opt }

func (f limitFilter) String() string {
	// ALL TABLES FROM WORKSPACE … --> EACH TABLES FROM WORKSPACE …
	// TAGS(…) --> EACH TAGS(…)
	const (
		all  = "ALL "
		each = "EACH "
	)
	s := fmt.Sprint(f.IFilter)
	if f.Option() == LimitFilterOption_EACH {
		if strings.HasPrefix(s, all) {
			s = strings.Replace(s, all, each, 1)
		} else {
			s = each + s
		}
	}
	return s
}

// # Supports:
//   - ILimit
type limit struct {
	typ
	ops  set.Set[OperationKind]
	opt  LimitFilterOption
	flt  ILimitFilter
	rate IRate
}

func newLimit(app *appDef, ws *workspace, name QName, ops []OperationKind, opt LimitFilterOption, flt IFilter, rate QName, comment ...string) *limit {
	if !LimitableOperations.ContainsAll(ops...) {
		panic(ErrUnsupported("limit operations %v", ops))
	}

	opSet := set.From(ops...)
	if compatible, err := IsCompatibleOperations(opSet); !compatible {
		panic(err)
	}
	if flt == nil {
		panic(ErrMissed("filter"))
	}
	l := &limit{
		typ:  makeType(app, ws, name, TypeKind_Limit),
		ops:  opSet,
		opt:  opt,
		flt:  newLimitFilter(opt, flt),
		rate: Rate(app.Type, rate),
	}
	if l.rate == nil {
		panic(ErrNotFound("rate «%v»", rate))
	}
	for t := range FilterMatches(l.Filter(), ws.Types()) {
		if err := l.validateOnType(t); err != nil {
			panic(err)
		}
	}
	l.typ.comment.setComment(comment...)
	ws.appendType(l)
	return l
}

func (l limit) Filter() ILimitFilter { return l.flt }

func (l limit) Op(o OperationKind) bool { return l.ops.Contains(o) }

func (l limit) Ops() iter.Seq[OperationKind] { return l.ops.Values() }

func (l limit) Rate() IRate { return l.rate }

// Validates limit.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not to be limited. See validateOnType
func (l limit) Validate() (err error) {
	cnt := 0
	for t := range FilterMatches(l.Filter(), l.Workspace().Types()) {
		err = errors.Join(err, l.validateOnType(t))
		cnt++
	}

	if (err == nil) && (cnt == 0) {
		return ErrFilterHasNoMatches(l, l.Filter(), l.Workspace())
	}

	return err
}

func (l limit) validateOnType(t IType) error {
	if !TypeKind_Limitables.Contains(t.Kind()) {
		return ErrUnsupported("%v can not to be limited", t)
	}
	return nil
}
