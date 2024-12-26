/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package rates

import (
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Supports:
//   - appdef.IRate
type Rate struct {
	types.Typ
	count  appdef.RateCount
	period appdef.RatePeriod
	scopes set.Set[appdef.RateScope]
}

func NewRate(ws appdef.IWorkspace, name appdef.QName, count appdef.RateCount, period appdef.RatePeriod, scopes []appdef.RateScope, comment ...string) *Rate {
	r := &Rate{
		Typ:    types.MakeType(ws.App(), ws, name, appdef.TypeKind_Rate),
		count:  count,
		period: period,
		scopes: set.From(scopes...),
	}
	if r.scopes.Len() == 0 {
		r.scopes.Set(appdef.DefaultRateScopes...)
	}
	comments.SetComment(r.Typ.WithComments, comment...)
	return r
}

func (r Rate) Count() appdef.RateCount { return r.count }

func (r Rate) Period() appdef.RatePeriod { return r.period }

func (r Rate) Scope(s appdef.RateScope) bool { return r.scopes.Contains(s) }

func (r Rate) Scopes() iter.Seq[appdef.RateScope] { return r.scopes.Values() }

// # Supports:
//   - appdef.ILimitFilter
type LimitFilter struct {
	appdef.IFilter
	opt appdef.LimitFilterOption
}

func NewLimitFilter(opt appdef.LimitFilterOption, flt appdef.IFilter) *LimitFilter {
	return &LimitFilter{flt, opt}
}

func (f LimitFilter) Option() appdef.LimitFilterOption { return f.opt }

func (f LimitFilter) String() string {
	// ALL TABLES FROM WORKSPACE … --> EACH TABLES FROM WORKSPACE …
	// TAGS(…) --> EACH TAGS(…)
	const (
		all  = "ALL "
		each = "EACH "
	)
	s := fmt.Sprint(f.IFilter)
	if f.Option() == appdef.LimitFilterOption_EACH {
		if strings.HasPrefix(s, all) {
			s = strings.Replace(s, all, each, 1)
		} else {
			s = each + s
		}
	}
	return s
}

// # Supports:
//   - appdef.ILimit
type Limit struct {
	types.Typ
	ops  set.Set[appdef.OperationKind]
	flt  appdef.ILimitFilter
	rate appdef.IRate
}

func NewLimit(ws appdef.IWorkspace, name appdef.QName, ops []appdef.OperationKind, opt appdef.LimitFilterOption, flt appdef.IFilter, rate appdef.QName, comment ...string) *Limit {
	if !appdef.LimitableOperations.ContainsAll(ops...) {
		panic(appdef.ErrUnsupported("limit operations %v", ops))
	}

	opSet := set.From(ops...)
	if compatible, err := appdef.IsCompatibleOperations(opSet); !compatible {
		panic(err)
	}
	if flt == nil {
		panic(appdef.ErrMissed("filter"))
	}
	l := &Limit{
		Typ:  types.MakeType(ws.App(), ws, name, appdef.TypeKind_Limit),
		ops:  opSet,
		flt:  NewLimitFilter(opt, flt),
		rate: appdef.Rate(ws.Type, rate),
	}
	if l.rate == nil {
		panic(appdef.ErrNotFound("rate «%v»", rate))
	}
	for t := range appdef.FilterMatches(l.Filter(), ws.Types()) {
		if err := l.validateOnType(t); err != nil {
			panic(err)
		}
	}
	comments.SetComment(l.Typ.WithComments, comment...)
	return l
}

func (l Limit) Filter() appdef.ILimitFilter { return l.flt }

func (l Limit) Op(o appdef.OperationKind) bool { return l.ops.Contains(o) }

func (l Limit) Ops() iter.Seq[appdef.OperationKind] { return l.ops.Values() }

func (l Limit) Rate() appdef.IRate { return l.rate }

// Validates limit.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not to be limited. See validateOnType
func (l Limit) Validate() (err error) {
	cnt := 0
	for t := range appdef.FilterMatches(l.Filter(), l.Workspace().Types()) {
		err = errors.Join(err, l.validateOnType(t))
		cnt++
	}

	if cnt == 0 {
		err = errors.Join(err, appdef.ErrFilterHasNoMatches(l, l.Filter(), l.Workspace()))
	}

	return err
}

func (l Limit) validateOnType(t appdef.IType) error {
	if !appdef.TypeKind_Limitables.Contains(t.Kind()) {
		return appdef.ErrUnsupported("%v can not to be limited", t)
	}
	return nil
}
