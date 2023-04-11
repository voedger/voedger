/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/voedger/pkg/iratesce"
	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/schemas"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

func TestRateLimits_BasicUsage(t *testing.T) {
	require := require.New(t)
	cfgs := make(AppConfigsType)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
	qName1 := istructs.NewQName(istructs.SysPackage, "myFunc")

	provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
	require.NoError(err)

	// limit c.sys.myFunc func call:
	// - per app:
	//   - not often than 10 times per day
	// - per workspace:
	//    - not often than once per minute

	// first - declare limits
	cfg.FunctionRateLimits.AddAppLimit(qName1, istructs.RateLimit{
		Period:                24 * time.Hour,
		MaxAllowedPerDuration: 10,
	})
	cfg.FunctionRateLimits.AddWorkspaceLimit(qName1, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 1,
	})

	// then - get AppStructs. For the first get default bucket states will be set
	as, err := provider.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)

	for i := 0; i < 10; i++ {
		// no limits exceeded
		require.False(as.IsFunctionRateLimitsExceeded(qName1, 42))

		// per-minute limit is exceeded
		require.True(as.IsFunctionRateLimitsExceeded(qName1, 42))

		// proceed to the next minute to restore per-minute limit
		coreutils.TestNow = coreutils.TestNow.Add(time.Minute)
	}

	// still failed because now the 10-hours limit is exceeded
	require.True(as.IsFunctionRateLimitsExceeded(qName1, 42))

	// try to add a minute the check if per-minute limit restore is not enough indeed
	coreutils.TestNow = coreutils.TestNow.Add(time.Minute)
	require.True(as.IsFunctionRateLimitsExceeded(qName1, 42))

	// add 10 hours to restore all limits
	coreutils.TestNow = coreutils.TestNow.Add(10 * time.Hour)
	require.False(as.IsFunctionRateLimitsExceeded(qName1, 42))

	t.Run("must be False if unknown (or unlimited) function", func(t *testing.T) {
		require.False(as.IsFunctionRateLimitsExceeded(istructs.NewQName("test", "unknown"), 42))
	})
}

func TestRateLimitsErrors(t *testing.T) {
	unsupportedRateLimitKind := istructs.RateLimitKind(istructs.RateLimitKind_FakeLast)
	rls := functionRateLimits{
		limits: map[istructs.QName]map[istructs.RateLimitKind]istructs.RateLimit{
			istructs.NewQName(istructs.SysPackage, "test"): {
				unsupportedRateLimitKind: {},
			},
		},
	}

	require.Panics(t, func() { rls.prepare(iratesce.Provide(time.Now)) })
}

func TestGetFunctionRateLimitName(t *testing.T) {

	testFn := istructs.NewQName(istructs.SysPackage, "test")

	tests := []struct {
		name string
		kind istructs.RateLimitKind
		want string
	}{
		{
			name: `RateLimitKind_byApp —> func_sys.test_ByApp`,
			kind: istructs.RateLimitKind_byApp,
			want: `func_sys.test_byApp`,
		},
		{
			name: `RateLimitKind_byWorkspace —> func_sys.test_ByWS`,
			kind: istructs.RateLimitKind_byWorkspace,
			want: `func_sys.test_byWS`,
		},
		{
			name: `RateLimitKind_byID —> func_sys.test_ByID`,
			kind: istructs.RateLimitKind_byID,
			want: `func_sys.test_byID`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFunctionRateLimitName(testFn, tt.kind); got != tt.want {
				t.Errorf("GetFunctionRateLimitName(%v, %v) = %v, want %v", testFn, tt.kind, got, tt.want)
			}
		})
	}

	t.Run("panic if kind is out of range", func(t *testing.T) {
		require.Panics(t, func() {
			_ = GetFunctionRateLimitName(testFn, istructs.RateLimitKind_FakeLast)
		})
	})
}
