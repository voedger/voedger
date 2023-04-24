/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

type functionRateLimits struct {
	limits map[schemas.QName]map[istructs.RateLimitKind]istructs.RateLimit
}

func (frl *functionRateLimits) addFuncLimit(funcQName schemas.QName) map[istructs.RateLimitKind]istructs.RateLimit {
	kindLimits, ok := frl.limits[funcQName]
	if !ok {
		kindLimits = map[istructs.RateLimitKind]istructs.RateLimit{}
		frl.limits[funcQName] = kindLimits
	}
	return kindLimits
}

func (frl *functionRateLimits) AddAppLimit(funcQName schemas.QName, rl istructs.RateLimit) {
	kindLimits := frl.addFuncLimit(funcQName)
	kindLimits[istructs.RateLimitKind_byApp] = rl
}

func (frl *functionRateLimits) AddWorkspaceLimit(funcQName schemas.QName, rl istructs.RateLimit) {
	kindLimits := frl.addFuncLimit(funcQName)
	kindLimits[istructs.RateLimitKind_byWorkspace] = rl
}

func (frl *functionRateLimits) prepare(buckets irates.IBuckets) {
	for funcQName, rls := range frl.limits {
		rateLimitName := ""
		for rlKind, rl := range rls {
			rateLimitName = GetFunctionRateLimitName(funcQName, rlKind)
			buckets.SetDefaultBucketState(rateLimitName, irates.BucketState{
				Period:             rl.Period,
				MaxTokensPerPeriod: irates.NumTokensType(rl.MaxAllowedPerDuration),
			})
		}
	}
}

func GetFunctionRateLimitName(funcQName schemas.QName, rateLimitKind istructs.RateLimitKind) (res string) {
	if rateLimitKind >= istructs.RateLimitKind_FakeLast {
		panic(fmt.Sprintf("unsupported limit kind %v", rateLimitKind))
	}
	return fmt.Sprintf(funcRateLimitNameFmt[rateLimitKind], funcQName)
}
