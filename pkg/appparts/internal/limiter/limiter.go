/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package limiter

import (
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

type Limiter struct {
	app     appdef.IAppDef
	buckets irates.IBuckets
	limits  map[appdef.QName][]appdef.ILimit
}

func New(app appdef.IAppDef, buckets irates.IBuckets) *Limiter {
	l := &Limiter{app, buckets, make(map[appdef.QName][]appdef.ILimit)}
	l.init()
	return l
}

// Return is specified resource (command, query or structure) usage limit is exceeded.
//
// If resource usage is exceeded then returns name of first exceeded limit.
func (l *Limiter) Exceeded(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) (bool, appdef.QName) {
	if limits, ok := l.limits[resource]; ok {
		keys := make([]irates.BucketKey, 0, len(limits))
		for _, limit := range limits {
			if limit.Op(operation) {
				key := irates.BucketKey{
					RateLimitName: limit.QName(),
				}
				if limit.Rate().Scope(appdef.RateScope_Workspace) {
					key.Workspace = workspace
				}
				if limit.Rate().Scope(appdef.RateScope_IP) {
					key.RemoteAddr = remoteAddr
				}
				if limit.Filter().Option() == appdef.LimitFilterOption_EACH {
					key.QName = resource
				}
				keys = append(keys, key)
			}
		}
		if len(keys) > 0 {
			ok, excLimit := l.buckets.TakeTokens(keys, 1)
			return !ok, excLimit
		}
	}

	return false, appdef.NullQName
}

func (l *Limiter) ResetLimits(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) {
	limits, ok := l.limits[resource]
	if !ok {
		return
	}
	for _, limit := range limits {
		if !limit.Op(operation) {
			continue
		}
		key := irates.BucketKey{
			RateLimitName: limit.QName(),
		}
		if limit.Rate().Scope(appdef.RateScope_Workspace) {
			key.Workspace = workspace
		}
		if limit.Rate().Scope(appdef.RateScope_IP) {
			key.RemoteAddr = remoteAddr
		}
		if limit.Filter().Option() == appdef.LimitFilterOption_EACH {
			key.QName = resource
		}
		defaultState, err := l.buckets.GetDefaultBucketsState(limit.QName())
		if err != nil {
			continue
		}
		err = l.buckets.SetBucketState(key, defaultState)
		if errors.Is(err, irates.ErrorRateLimitNotFound) {
			// SetBucketState returns ErrorRateLimitNotFound when no bucket exists for the key.
			// In the production IBuckets implementation (iratesce.bucketsType), bucketByKey
			// auto-creates a bucket when a default state exists for the RateLimitName.
			// Since Limiter.init() registers default states for all limits, and we iterate
			// only over those limits here, this error cannot occur in practice.
			// For alternative IBuckets implementations: resetting a bucket that was never
			// used (no tokens taken) is a no-op by definition, so skipping is safe.
			continue
		}
		if err != nil {
			// notest
			panic(err)
		}
	}
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

	// initialize limits cache
	for _, t := range l.app.Types() {
		if appdef.TypeKind_Limitables.Contains(t.Kind()) {
			for limit := range appdef.Limits(l.app.Types()) {
				if limit.Filter().Match(t) {
					l.limits[t.QName()] = append(l.limits[t.QName()], limit)
				}
			}
		}
	}
}
