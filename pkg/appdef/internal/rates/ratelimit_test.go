/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package rates_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestRateLimits(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	cmdName := appdef.NewQName("test", "command")
	tagName := appdef.NewQName("test", "tag")
	rateName := appdef.NewQName("test", "rate")
	limitAllFunc := appdef.NewQName("test", "limitAllFunc")
	limitEachFunc := appdef.NewQName("test", "limitEachFunc")
	limitEachTag := appdef.NewQName("test", "limitEachTag")

	t.Run("should be ok to build application with rates and limits", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagName)
		wsb.AddCommand(cmdName).SetTag(tagName)

		wsb.AddRate(rateName, 10, time.Hour, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "10 times per hour per partition per IP")
		wsb.AddLimit(limitAllFunc, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, filter.AllWSFunctions(wsName), rateName, "limit all commands and queries execution by test.rate")
		wsb.AddLimit(limitEachFunc, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_EACH, filter.AllWSFunctions(wsName), rateName, "limit each command and query execution by test.rate")
		wsb.AddLimit(limitEachTag, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_EACH, filter.Tags(tagName), rateName, "limit each with test.tag by test.rate")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to enum rates", func(t *testing.T) {
		cnt := 0
		for r := range appdef.Rates(app.Types()) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(rateName, r.QName())
				require.EqualValues(10, r.Count())
				require.Equal(time.Hour, r.Period())

				require.Equal([]appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, r.Scopes())
				require.True(r.Scope(appdef.RateScope_AppPartition))
				require.True(r.Scope(appdef.RateScope_IP))
				require.False(r.Scope(appdef.RateScope_Workspace))

				require.Equal("10 times per hour per partition per IP", r.Comment())
			default:
				require.FailNow("unexpected rate", "rate: %v", r)
			}
		}
		require.Equal(1, cnt)
	})

	t.Run("should be ok to enum limits", func(t *testing.T) {
		cnt := 0
		for l := range appdef.Limits(app.Types()) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(limitAllFunc, l.QName())

				require.Equal([]appdef.OperationKind{appdef.OperationKind_Execute}, l.Ops())
				require.True(l.Op(appdef.OperationKind_Execute))
				require.False(l.Op(appdef.OperationKind_Insert))

				require.Equal(appdef.FilterKind_Types, l.Filter().Kind())
				require.Equal(appdef.LimitFilterOption_ALL, l.Filter().Option())
				require.Equal([]appdef.TypeKind{appdef.TypeKind_Query, appdef.TypeKind_Command}, l.Filter().Types())
				require.Equal("ALL FUNCTIONS FROM test.workspace", fmt.Sprint(l.Filter()))

				require.Equal(rateName, l.Rate().QName())

				require.Equal("limit all commands and queries execution by test.rate", l.Comment())
			case 2:
				require.Equal(limitEachFunc, l.QName())
				require.Equal([]appdef.OperationKind{appdef.OperationKind_Execute}, l.Ops())
				require.Equal(appdef.FilterKind_Types, l.Filter().Kind())
				require.Equal(appdef.LimitFilterOption_EACH, l.Filter().Option())
				require.Equal([]appdef.TypeKind{appdef.TypeKind_Query, appdef.TypeKind_Command}, l.Filter().Types())
				require.Equal("EACH FUNCTIONS FROM test.workspace", fmt.Sprint(l.Filter()))
				require.Equal(rateName, l.Rate().QName())
				require.Equal("limit each command and query execution by test.rate", l.Comment())
			case 3:
				require.Equal(limitEachTag, l.QName())
				require.Equal([]appdef.OperationKind{appdef.OperationKind_Execute}, l.Ops())
				require.Equal(appdef.FilterKind_Tags, l.Filter().Kind())
				require.Equal(appdef.LimitFilterOption_EACH, l.Filter().Option())
				require.Equal([]appdef.QName{tagName}, l.Filter().Tags())
				require.Equal("EACH TAGS(test.tag)", fmt.Sprint(l.Filter()))
				require.Equal(rateName, l.Rate().QName())
				require.Equal("limit each with test.tag by test.rate", l.Comment())
			default:
				require.FailNow("unexpected limit", "limit: %v", l)
			}
		}
		require.Equal(3, cnt)
	})

	t.Run("should be ok to find rates and limits", func(t *testing.T) {
		unknown := appdef.NewQName("test", "unknown")

		rate := appdef.Rate(app.Type, rateName)
		require.NotNil(rate)
		require.Equal(rateName, rate.QName())

		require.Nil(appdef.Rate(app.Type, unknown), "should be nil if unknown")

		limit := appdef.Limit(app.Type, limitAllFunc)
		require.NotNil(limit)
		require.Equal(limitAllFunc, limit.QName())

		require.Nil(appdef.Limit(app.Type, unknown), "should be nil if unknown")
	})

	t.Run("should be ok to add rate with default scope", func(t *testing.T) {
		app := func() appdef.IAppDef {
			rateName := appdef.NewQName("test", "rate")
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")

			return adb.MustBuild()
		}()

		r := appdef.Rate(app.Type, rateName)
		require.Equal(rateName, r.QName())
		require.EqualValues(10, r.Count())
		require.Equal(time.Hour, r.Period())
		require.Equal(appdef.DefaultRateScopes, r.Scopes())
		require.Equal("10 times per hour", r.Comment())
	})
}

func Test_RateLimitErrors(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	rateName := appdef.NewQName("test", "rate")
	limitName := appdef.NewQName("test", "limit")

	unknown := appdef.NewQName("test", "unknown")

	t.Run("should panics", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")

		t.Run("if missed operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName,
					[]appdef.OperationKind{}, // <-- missed operations
					appdef.LimitFilterOption_ALL, filter.AllWSTables(wsName), rateName)
			},
				require.Is(appdef.ErrMissedError), require.Has("operations"))
		})

		t.Run("if not limitable operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName,
					[]appdef.OperationKind{appdef.OperationKind_Inherits}, // <-- non limitable operation
					appdef.LimitFilterOption_ALL, filter.AllWSTables(wsName), rateName)
			},
				require.Is(appdef.ErrUnsupportedError), require.Has("Inherits"))
		})

		t.Run("if incompatible operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName,
					[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Execute}, // <-- incompatible operations
					appdef.LimitFilterOption_ALL, filter.AllWSTables(wsName), rateName)
			},
				require.Is(appdef.ErrIncompatibleError), require.Has("operations"))
		})

		t.Run("if missed filter", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName,
					[]appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL,
					nil, // <-- missed filter
					rateName)
			},
				require.Is(appdef.ErrMissedError), require.Has("filter"))
		})

		t.Run("if filtered object is not limitable", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL,
					filter.QNames(appdef.SysData_bool), // <-- not limitable
					rateName)
			},
				require.Is(appdef.ErrUnsupportedError), require.Has(appdef.SysData_bool))
		})

		t.Run("if missed or unknown rate", func(t *testing.T) {
			require.Panics(func() {
				wsb.AddLimit(limitName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, filter.AllWSFunctions(wsName),
					unknown, // <-- unknown rate
				)
			},
				require.Is(appdef.ErrNotFoundError), require.Has(unknown))
		})
	})

	t.Run("should be validate error", func(t *testing.T) {

		t.Run("if no types matched", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")
			f := filter.AllWSFunctions(wsName)
			wsb.AddLimit(limitName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, f, rateName)

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.HasAll(f, "no matches", wsName))
		})

		t.Run("if filtered object is not limitable", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			testName := appdef.NewQName("test", "test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddRate(rateName, 10, time.Hour, nil, "10 times per hour")
			wsb.AddLimit(limitName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL,
				filter.QNames(testName), // <-- not limitable
				rateName)

			_ = wsb.AddRole(testName)

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrUnsupportedError), require.Has(testName))
		})
	})
}

func TestLimitActivateDeactivate(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "document")
	rateName := appdef.NewQName("test", "rate")
	limitDeactivate := appdef.NewQName("test", "limitDeactivate")
	limitActivate := appdef.NewQName("test", "limitActivate")

	t.Run("should be ok to build application with activate/deactivate limits", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddCDoc(docName)

		wsb.AddRate(rateName, 10, time.Hour, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "10 times per hour per partition per IP")
		wsb.AddLimit(limitDeactivate, []appdef.OperationKind{appdef.OperationKind_Deactivate}, appdef.LimitFilterOption_EACH, filter.QNames(docName), rateName)
		wsb.AddLimit(limitActivate, []appdef.OperationKind{appdef.OperationKind_Activate}, appdef.LimitFilterOption_EACH, filter.QNames(docName), rateName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to enum limits", func(t *testing.T) {
		cnt := 0
		for l := range appdef.Limits(app.Types()) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(limitActivate, l.QName())
				require.Equal([]appdef.OperationKind{appdef.OperationKind_Activate}, l.Ops())
				require.Equal(appdef.FilterKind_QNames, l.Filter().Kind())
				require.Equal(appdef.LimitFilterOption_EACH, l.Filter().Option())
				require.Equal([]appdef.QName{docName}, l.Filter().QNames())
				require.Equal(rateName, l.Rate().QName())
			case 2:
				require.Equal(limitDeactivate, l.QName())
				require.Equal([]appdef.OperationKind{appdef.OperationKind_Deactivate}, l.Ops())
				require.Equal(appdef.FilterKind_QNames, l.Filter().Kind())
				require.Equal(appdef.LimitFilterOption_EACH, l.Filter().Option())
				require.Equal([]appdef.QName{docName}, l.Filter().QNames())
				require.Equal(rateName, l.Rate().QName())
			default:
				require.FailNow("unexpected limit", "limit: %v", l)
			}
		}
		require.Equal(2, cnt)
	})
}
