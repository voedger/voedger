/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructsmem

import (
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestRateLimits_BasicUsage(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	cfgs := make(AppConfigsType)
	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	qName1 := appdef.NewQName("test", "myFunc")

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

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
	as, err := provider.BuiltIn(appName)
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
		require.False(as.IsFunctionRateLimitsExceeded(appdef.NewQName("test", "unknown"), 42))
	})
}

func TestRateLimitsErrors(t *testing.T) {
	require := require.New(t)
	unsupportedRateLimitKind := istructs.RateLimitKind_FakeLast
	rls := functionRateLimits{
		limits: map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit{
			appdef.NewQName(appdef.SysPackage, "test"): {
				unsupportedRateLimitKind: {},
			},
		},
	}

	require.Panics(func() { rls.prepare(iratesce.Provide(time.Now)) },
		require.Has(unsupportedRateLimitKind))
}

func TestGetFunctionRateLimitName(t *testing.T) {

	testFn := appdef.NewQName(appdef.SysPackage, "test")

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
		require := require.New(t)
		require.Panics(func() {
			_ = GetFunctionRateLimitName(testFn, istructs.RateLimitKind_FakeLast)
		}, require.Has(istructs.RateLimitKind_FakeLast))
	})
}
