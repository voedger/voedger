/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package processors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestWithEventLogAttrs(t *testing.T) {
	for _, level := range []logger.TLogLevel{logger.LogLevelVerbose, logger.LogLevelInfo} {
		t.Run(fmtLogLevel(level), func(t *testing.T) {
			logs := logger.StartCapture(t, level)
			event := &coreutils.MockPLogEvent{}
			event.On("WLogOffset").Return(istructs.Offset(7))
			event.On("QName").Return(appdef.NewQName("test", "Command"))
			ctx := logger.WithContextAttrs(t.Context(), map[string]any{"vapp": "test/app", "wsid": 42, "extension": "ap.test.Projector"})
			ctx = WithEventLogAttrs(ctx, event, 19)
			logs.NotContains("stage=")
			logger.ErrorCtx(ctx, "ap.error", "failed")
			logs.HasLine("stage=ap.error", "vapp=test/app", "wsid=42", "extension=ap.test.Projector",
				"woffset=7", "poffset=19", "evqname=test.Command", "msg=failed")
			logs.NotContains("args=")
			event.AssertNotCalled(t, "ArgumentObject")
			event.AssertExpectations(t)
		})
	}
}

func fmtLogLevel(level logger.TLogLevel) string {
	if level == logger.LogLevelVerbose {
		return "verbose"
	}
	return "info"
}

func TestEventLoggingArguments(t *testing.T) {
	for _, withArgs := range []bool{true, false} {
		name := "without arguments"
		if withArgs {
			name = "existing default includes arguments"
		}
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			logs := logger.StartCapture(t, logger.LogLevelVerbose)
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(appdef.NewQName("test", "Workspace"))
			argName, rowName := appdef.NewQName("test", "Args"), appdef.NewQName("test", "Row")
			ws.AddObject(argName).AddField("value", appdef.DataKind_string, false)
			ws.AddCDoc(rowName)
			app := adb.MustBuild()
			event := &coreutils.MockPLogEvent{}
			event.On("WLogOffset").Return(istructs.Offset(7))
			event.On("QName").Return(appdef.NewQName("test", "Command"))
			if withArgs {
				event.On("ArgumentObject").Return(&coreutils.TestObject{Name: argName, Data: map[string]any{appdef.SystemField_QName: argName, "value": "payload"}})
			}
			kept := &coreutils.TestObject{Name: rowName, ID_: 1, IsNew_: true}
			skipped := &coreutils.TestObject{Name: rowName, ID_: 2, IsNew_: true}
			event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
				cb := args.Get(0).(func(istructs.ICUDRow) bool)
				if cb(kept) {
					cb(skipped)
				}
			})
			filterCUD := func(row istructs.ICUDRow) (bool, string, error) { return row.ID() == 1, "", nil }
			logEvent := LogEventAndCUDsWithoutArgs
			if withArgs {
				logEvent = LogEventAndCUDs
			}
			ctx, err := logEvent(t.Context(), event, 19, app, 0, "ap", filterCUD, "triggeredby=test.Command")
			require.NoError(err)
			require.NotNil(ctx)
			if withArgs {
				logs.HasLine("stage=ap ", "args=", "payload", "triggeredby=test.Command")
			} else {
				logs.HasLine("stage=ap ", `msg="triggeredby=test.Command"`)
				logs.NotContains("args=", "payload")
				event.AssertNotCalled(t, "ArgumentObject")
			}
			logs.HasLine("stage=ap.log_cud", "recid=1", "op=create")
			logs.NotContains("recid=2")
			event.AssertExpectations(t)
		})
	}
	t.Run("disabled verbose logging does not inspect event", func(t *testing.T) {
		require := require.New(t)
		logs := logger.StartCapture(t, logger.LogLevelInfo)
		for _, logEvent := range []func(context.Context, istructs.IPLogEvent, istructs.Offset, appdef.IAppDef, int, string, func(istructs.ICUDRow) (bool, string, error), string) (context.Context, error){LogEventAndCUDs, LogEventAndCUDsWithoutArgs} {
			ctx := t.Context()
			actual, err := logEvent(ctx, nil, 0, nil, 0, "ap", nil, "")
			require.NoError(err)
			require.Equal(ctx, actual)
		}
		logs.NotContains("stage=")
	})
	t.Run("CUD callback failure propagates", func(t *testing.T) {
		require := require.New(t)
		logger.StartCapture(t, logger.LogLevelVerbose)
		event := &coreutils.MockPLogEvent{}
		event.On("WLogOffset").Return(istructs.Offset(7))
		event.On("QName").Return(appdef.NewQName("test", "Command"))
		event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
			args.Get(0).(func(istructs.ICUDRow) bool)(&coreutils.TestObject{})
		})
		expected := errors.New("CUD failure")
		_, err := LogEventAndCUDsWithoutArgs(t.Context(), event, 19, nil, 0, "ap",
			func(istructs.ICUDRow) (bool, string, error) { return false, "", expected }, "")
		require.ErrorIs(err, expected)
		event.AssertNotCalled(t, "ArgumentObject")
	})
}

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
