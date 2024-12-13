/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"slices"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDefAddRateLimit(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	cmdName := appdef.NewQName("test", "command")
	rateName := appdef.NewQName("test", "rate")
	limitName := appdef.NewQName("test", "limit")

	t.Run("should be ok to build application with rates and limits", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)

		wsb.AddRate(rateName, 10, time.Hour, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "10 times per hour per partition per IP")
		wsb.AddLimit(limitName, appdef.LimitOption_ALL, []appdef.OperationKind{appdef.OperationKind_Execute}, filter.AllFunctions(wsName), rateName, "limit all commands and queries execution with test.rate")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to enum rates", func(t *testing.T) {
			cnt := 0
			for r := range appdef.Rates(tested.Types()) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(rateName, r.QName())
					require.EqualValues(10, r.Count())
					require.Equal(time.Hour, r.Period())
					require.Equal([]appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, slices.Collect(r.Scopes()))
					require.Equal("10 times per hour per partition per IP", r.Comment())
				default:
					require.FailNow("unexpected rate", "rate: %v", r)
				}
			}
			require.Equal(1, cnt)
		})

		t.Run("should be ok to enum limits", func(t *testing.T) {
			cnt := 0
			for l := range appdef.Limits(tested.Types()) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(limitName, l.QName())
					require.Equal(appdef.LimitOption_ALL, l.Option())
					require.Equal([]appdef.OperationKind{appdef.OperationKind_Execute}, slices.Collect(l.Ops()))
					require.True(l.Op(appdef.OperationKind_Execute))
					require.False(l.Op(appdef.OperationKind_Insert))
					require.Equal(appdef.FilterKind_Types, l.Filter().Kind())
					require.Equal([]appdef.TypeKind{appdef.TypeKind_Query, appdef.TypeKind_Command}, slices.Collect(l.Filter().Types()))
					require.Equal(rateName, l.Rate().QName())
					require.Equal("limit all commands and queries execution with test.rate", l.Comment())
				default:
					require.FailNow("unexpected limit", "limit: %v", l)
				}
			}
		})

		t.Run("should be ok to find rates and limits", func(t *testing.T) {
			unknown := appdef.NewQName("test", "unknown")

			rate := appdef.Rate(tested.Type, rateName)
			require.NotNil(rate)
			require.Equal(rateName, rate.QName())

			require.Nil(appdef.Rate(tested.Type, unknown), "should be nil if unknown")

			limit := appdef.Limit(tested.Type, limitName)
			require.NotNil(limit)
			require.Equal(limitName, limit.QName())

			require.Nil(appdef.Limit(tested.Type, unknown), "should be nil if unknown")
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	t.Run("should be ok to add rate with default scope", func(t *testing.T) {
		app := func() appdef.IAppDef {
			rateName := appdef.NewQName("test", "rate")
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")

			return adb.MustBuild()
		}()

		r := appdef.Rate(app.Type, rateName)
		require.Equal(rateName, r.QName())
		require.EqualValues(10, r.Count())
		require.Equal(time.Hour, r.Period())
		require.Equal(appdef.DefaultRateScopes, slices.Collect(r.Scopes()))
		require.Equal("10 times per hour", r.Comment())
	})
}

func Test_AppDefAddRateLimitErrors(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	rateName := appdef.NewQName("test", "rate")
	limitName := appdef.NewQName("test", "limit")

	unknown := appdef.NewQName("test", "unknown")

	t.Run("should panics", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")

		t.Run("if missed operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, appdef.LimitOption_ALL,
					[]appdef.OperationKind{}, // <-- missed operations
					filter.AllTables(wsName), rateName)
			},
				require.Is(appdef.ErrMissedError), require.Has("operations"))
		})

		t.Run("if incompatible operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, appdef.LimitOption_ALL,
					[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Execute}, // <-- incompatible operations
					filter.AllTables(wsName), rateName)
			},
				require.Is(appdef.ErrIncompatibleError), require.Has("operations"))
		})

		t.Run("if missed filter", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, appdef.LimitOption_ALL,
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					nil, // <-- missed filter
					rateName)
			},
				require.Is(appdef.ErrMissedError), require.Has("filter"))
		})

		t.Run("if filtered object is not limitable", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, appdef.LimitOption_ALL, []appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(appdef.SysData_bool), // <-- not limitable
					rateName)
			},
				require.Is(appdef.ErrUnsupportedError), require.Has(appdef.SysData_bool))
		})

		t.Run("if missed or unknown rate", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, appdef.LimitOption_ALL, []appdef.OperationKind{appdef.OperationKind_Execute}, filter.AllFunctions(wsName),
					unknown, // <-- unknown rate
				)
			},
				require.Is(appdef.ErrNotFoundError), require.Has(unknown))
		})
	})

	t.Run("should be validate error", func(t *testing.T) {

		t.Run("if no types matched", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")
			f := filter.AllFunctions(wsName)
			wsb.AddLimit(limitName, appdef.LimitOption_ALL, []appdef.OperationKind{appdef.OperationKind_Execute}, f, rateName)

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.HasAll(f, "no matches", wsName))
		})

		t.Run("if filtered object is not limitable", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			testName := appdef.NewQName("test", "test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")
			wsb.AddLimit(limitName, appdef.LimitOption_ALL, []appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testName), // <-- not limitable
				rateName)

			_ = wsb.AddRole(testName)

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrUnsupportedError), require.Has(testName))
		})
	})
}
