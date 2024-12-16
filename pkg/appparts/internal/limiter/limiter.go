/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package limiter

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

type Limiter struct {
	app     appdef.IAppDef
	buckets irates.IBuckets
	objects map[appdef.QName][]appdef.ILimit
}

func newLimiter(app appdef.IAppDef, buckets irates.IBuckets) *Limiter {
	l := &Limiter{app, buckets, make(map[appdef.QName][]appdef.ILimit)}
	l.init()
	return l
}

// Return is specified resource (command, query or structure) usage limit is exceeded.
//
// If resource usage is exceeded then returns name of first exceeded limit.
func (l *Limiter) Exceeded(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) (bool, appdef.QName) {
	return false, appdef.NullQName
}

func (l *Limiter) init() {
	// initialize default buckets states
	for limit := range appdef.Limits(l.app.Types()) {
		l.buckets.SetDefaultBucketState(
			limit.QName(),
			irates.BucketState{
				Period:             limit.Rate().Period(),
				MaxTokensPerPeriod: limit.Rate().Count(),
			})
	}

	// initialize objects cache
	for t := range l.app.Types() {
		if appdef.TypeKind_Limitables.Contains(t.Kind()) {
			for limit := range appdef.Limits(l.app.Types()) {
				if limit.Filter().Match(t) {
					l.objects[t.QName()] = append(l.objects[t.QName()], limit)
				}
			}
		}
	}
}
