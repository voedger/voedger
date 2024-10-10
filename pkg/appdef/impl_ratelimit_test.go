/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDefAddRateLimit(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	rateName := NewQName("test", "rate")
	limitName := NewQName("test", "limit")

	t.Run("should be ok to build application with rates and limits", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		adb.AddRate(rateName, 10, time.Hour, []RateScope{RateScope_AppPartition, RateScope_IP}, "10 times per hour per partition per IP")
		adb.AddLimit(limitName, []QName{QNameAnyFunction}, rateName, "limit all commands and queries execution with test.rate")

		app = adb.MustBuild()
	})

	t.Run("should be ok to enum rates", func(t *testing.T) {
		cnt := 0
		for r := range app.Rates {
			cnt++
			switch cnt {
			case 1:
				require.Equal(rateName, r.QName())
				require.EqualValues(10, r.Count())
				require.Equal(time.Hour, r.Period())
				require.Equal([]RateScope{RateScope_AppPartition, RateScope_IP}, r.Scopes())
				require.Equal("10 times per hour per partition per IP", r.Comment())
			default:
				require.FailNow("unexpected rate", "rate: %v", r)
			}
		}
		require.Equal(1, cnt)
	})

	t.Run("should be ok to enum limits", func(t *testing.T) {
		cnt := 0
		for l := range app.Limits {
			cnt++
			switch cnt {
			case 1:
				require.Equal(limitName, l.QName())
				require.Equal(QNamesFrom(QNameAnyFunction), l.On())
				require.Equal(rateName, l.Rate().QName())
				require.Equal("limit all commands and queries execution with test.rate", l.Comment())
			default:
				require.FailNow("unexpected limit", "limit: %v", l)
			}
		}
	})

	t.Run("should be ok to find rates and limits", func(t *testing.T) {
		rate := app.Rate(rateName)
		require.NotNil(rate)
		require.Equal(rateName, rate.QName())

		require.Nil(app.Rate(NewQName("test", "unknown")), "should be nil if unknown rate")

		limit := app.Limit(limitName)
		require.NotNil(limit)
		require.Equal(limitName, limit.QName())

		require.Nil(app.Limit(NewQName("test", "unknown")), "should be nil if unknown limit")
	})

	t.Run("should be ok to add rate with default scope", func(t *testing.T) {
		app := func() IAppDef {
			rateName := NewQName("test", "rate")
			adb := New()
			adb.AddPackage("test", "test.com/test")

			adb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")

			return adb.MustBuild()
		}()

		r := app.Rate(rateName)
		require.Equal(rateName, r.QName())
		require.EqualValues(10, r.Count())
		require.Equal(time.Hour, r.Period())
		require.Equal(DefaultRateScopes, r.Scopes())
		require.Equal("10 times per hour", r.Comment())
	})
}

func Test_AppDefAddRateLimitErrors(t *testing.T) {
	require := require.New(t)

	rateName := NewQName("test", "rate")
	limitName := NewQName("test", "limit")

	t.Run("should panic if missed objects", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		adb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")

		require.Panics(func() { adb.AddLimit(limitName, nil, rateName) },
			require.Is(ErrMissedError))
	})

	t.Run("should panic if missed or unknown rate", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		require.Panics(func() { adb.AddLimit(limitName, []QName{QNameAnyCommand}, NullQName) },
			require.Is(ErrMissedError))
		require.Panics(func() { adb.AddLimit(limitName, []QName{QNameAnyCommand}, NewQName("test", "unknown")) },
			require.Is(ErrNotFoundError), require.Has("test.unknown"))
	})

	t.Run("test validate errors", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		adb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")
		adb.AddLimit(limitName, []QName{NewQName("test", "unknown"), QNameAnyCommand}, rateName)

		_, err := adb.Build()

		require.Error(err, require.Is(ErrNotFoundError), require.Has("test.unknown"))
	})
}
