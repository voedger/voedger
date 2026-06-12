/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package processors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
)

func TestRetryAfterSecondsOnLimitExceeded(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	cmdName := appdef.NewQName("test", "cmd")
	buildApp := func(count appdef.RateCount, period time.Duration) (appdef.IAppDef, appdef.QName) {
		limitName := appdef.NewQName("test", "limit")
		rateName := appdef.NewQName("test", "rate")
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCommand(cmdName)
		wsb.AddRate(rateName, count, period, nil)
		wsb.AddLimit(limitName,
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			appdef.LimitFilterOption_EACH,
			filter.WSTypes(wsName, appdef.TypeKind_Command),
			rateName)
		return adb.MustBuild(), limitName
	}

	t.Run("integer seconds", func(t *testing.T) {
		app, limit := buildApp(6, time.Minute) // 60s / 6 = 10s
		require.Equal(10, RetryAfterSecondsOnLimitExceeded(app, limit))
	})

	t.Run("sub-second per token rounds up to 1", func(t *testing.T) {
		app, limit := buildApp(1000, time.Minute) // 60s / 1000 = 0.06s -> 1
		require.Equal(1, RetryAfterSecondsOnLimitExceeded(app, limit))
	})

	t.Run("fractional seconds rounded up", func(t *testing.T) {
		app, limit := buildApp(7, time.Minute) // 60/7 = 8.57... -> 9
		require.Equal(9, RetryAfterSecondsOnLimitExceeded(app, limit))
	})

	t.Run("long period", func(t *testing.T) {
		app, limit := buildApp(1, time.Hour) // 3600s / 1 = 3600
		require.Equal(3600, RetryAfterSecondsOnLimitExceeded(app, limit))
	})
}
