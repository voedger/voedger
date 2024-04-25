/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"time"
)

// Implements:
//   - IRate
type rate struct {
	typ
	count  int
	period time.Duration
	scopes RateScopes
}

func newRate(app *appDef, name QName, count int, period time.Duration, scopes []RateScope) *rate {
	r := &rate{
		typ:    makeType(app, name, TypeKind_Rate),
		count:  count,
		period: period,
		scopes: scopes,
	}
	app.appendType(r)
	return r
}

func (r rate) Count() int {
	return r.count
}

func (r rate) Period() time.Duration {
	return r.period
}

func (r rate) Scopes() RateScopes {
	return r.scopes
}

func (r rate) String() string {
	return fmt.Sprintf("%v %d per %v per %v", r.typ, r.count, r.period, r.scopes)
}
