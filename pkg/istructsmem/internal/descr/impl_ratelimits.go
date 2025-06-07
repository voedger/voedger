/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newRate() *Rate {
	return &Rate{}
}

func (r *Rate) read(rate appdef.IRate) {
	r.Type.read(rate)
	r.Count = rate.Count()
	r.Period = rate.Period()
	for _, scope := range rate.Scopes() {
		r.Scopes = append(r.Scopes, scope.TrimString())
	}
}

func newLimit() *Limit {
	return &Limit{}
}

func (l *Limit) read(limit appdef.ILimit) {
	l.Type.read(limit)
	for _, op := range limit.Ops() {
		l.Ops = append(l.Ops, op.TrimString())
	}
	l.Filter.read(limit.Filter())
	l.Rate = limit.Rate().QName()
}

func (f *LimitFilter) read(flt appdef.ILimitFilter) {
	f.Option = flt.Option().TrimString()
	f.Filter.read(flt)
}
