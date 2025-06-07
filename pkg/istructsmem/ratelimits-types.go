/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

type functionRateLimits struct {
	limits map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit
}

func (frl *functionRateLimits) addFuncLimit(funcQName appdef.QName) map[istructs.RateLimitKind]istructs.RateLimit {
	kindLimits, ok := frl.limits[funcQName]
	if !ok {
		kindLimits = map[istructs.RateLimitKind]istructs.RateLimit{}
		frl.limits[funcQName] = kindLimits
	}
	return kindLimits
}

func (frl *functionRateLimits) AddAppLimit(funcQName appdef.QName, rl istructs.RateLimit) {
	kindLimits := frl.addFuncLimit(funcQName)
	kindLimits[istructs.RateLimitKind_byApp] = rl
}

func (frl *functionRateLimits) AddWorkspaceLimit(funcQName appdef.QName, rl istructs.RateLimit) {
	kindLimits := frl.addFuncLimit(funcQName)
	kindLimits[istructs.RateLimitKind_byWorkspace] = rl
}

func (frl *functionRateLimits) prepare(buckets irates.IBuckets) {
	for funcQName, rls := range frl.limits {
		var rateLimitName appdef.QName
		for rlKind, rl := range rls {
			rateLimitName = GetFunctionRateLimitName(funcQName, rlKind)
			buckets.SetDefaultBucketState(rateLimitName, irates.BucketState{
				Period:             rl.Period,
				MaxTokensPerPeriod: rl.MaxAllowedPerDuration,
			})
		}
	}
}

func GetFunctionRateLimitName(funcQName appdef.QName, rateLimitKind istructs.RateLimitKind) (res appdef.QName) {
	if rateLimitKind >= istructs.RateLimitKind_FakeLast {
		panic(fmt.Sprintf("unsupported limit kind %v", rateLimitKind))
	}
	return appdef.NewQName(
		funcQName.Pkg(),
		fmt.Sprintf(funcRateLimitNameFmt[rateLimitKind], funcQName.Entity()),
	)
}
