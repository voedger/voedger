/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package logger_test

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func Test_SetLogLevelWithRestore(t *testing.T) {
	require := require.New(t)
	func() {
		defer logger.SetLogLevelWithRestore(logger.LogLevelTrace)()
		require.True(logger.IsTrace())
	}()
	require.False(logger.IsTrace())
}

func TestLegacyFuncs_BasicUsage(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	logger.Verbose("verbose message")
	logger.Info("info message")
	logger.Warning("warning message")
	logger.Error("error message")
	logCap.HasLine("verbose message")
	logCap.HasLine("info message")
	logCap.HasLine("warning message")
	logCap.HasLine("error message")
}

func TestSlogFuncs_BasicUsage(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	ctx := logger.WithContextAttrs(context.Background(), map[string]any{
		logger.LogAttr_VApp: "myapp",
		logger.LogAttr_WSID: 42,
	})
	logger.VerboseCtx(ctx, "", "verbose ctx message")
	logger.InfoCtx(ctx, "", "info ctx message")
	logger.ErrorCtx(ctx, "", "error ctx message")
	logger.LogCtx(ctx, 0, logger.LogLevelInfo, "", "log ctx message")
	logCap.HasLine("verbose ctx message", "vapp=myapp", "wsid=42")
	logCap.HasLine("info ctx message", "vapp=myapp")
	logCap.HasLine("error ctx message", "wsid=42")
	logCap.HasLine("log ctx message", "vapp=myapp", "wsid=42")
}

func loggerHelperWithSkipStackFrames(skipStackFrames int, msg string) error {
	logger.Log(skipStackFrames, logger.LogLevelTrace, "myStunningPrefix:", msg)
	return nil
}

func Test_BasicUsage_CustomPrintLine(t *testing.T) {
	require := require.New(t)

	myPrintLine := func(level logger.TLogLevel, line string) {
		line += "myPrintLine"
		logger.DefaultPrintLine(level, line)
	}

	logger.PrintLine = myPrintLine
	defer func() { logger.PrintLine = logger.DefaultPrintLine }()

	logCap := logger.StartCapture(t, logger.LogLevelTrace)
	logger.Trace("hello")
	require.Contains(logCap.String(), "myPrintLine")
}

func Test_SkipStackFrames(t *testing.T) {
	const funcNamePattern = "loggerHelperWithSkipStackFrames"

	t.Run("skipStackFrames=0 shows helper func name", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelTrace)
		_ = loggerHelperWithSkipStackFrames(0, "hello")
		logCap.HasLine(funcNamePattern)
	})

	t.Run("skipStackFrames=1 hides helper func name", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelTrace)
		_ = loggerHelperWithSkipStackFrames(1, "hello")
		logCap.NotContains(funcNamePattern)
	})
}

func Test_StdoutStderr_LogLevel(t *testing.T) {
	t.Run("LogLevelError: Error visible, Warning suppressed", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelError)
		logger.Error("Error", "arg1", "arg2")
		logger.Warning("My warning")
		logCap.HasLine("Error arg1 arg2")
		logCap.NotContains("My warning")
	})

	t.Run("LogLevelWarning: Error and Warning both visible", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelWarning)
		logger.Error("Error", "arg1", "arg2")
		logger.Warning("My warning")
		logCap.HasLine("Error arg1 arg2")
		logCap.HasLine("My warning")
	})
}

func Test_CheckSetLevels(t *testing.T) {
	allChecks := []func() bool{logger.IsError, logger.IsWarning, logger.IsInfo, logger.IsVerbose, logger.IsTrace}
	testCases := []struct {
		name      string
		level     logger.TLogLevel
		activeIdx int
	}{
		{"None", logger.LogLevelNone, -1},
		{"Error", logger.LogLevelError, 0},
		{"Warning", logger.LogLevelWarning, 1},
		{"Info", logger.LogLevelInfo, 2},
		{"Verbose", logger.LogLevelVerbose, 3},
		{"Trace", logger.LogLevelTrace, 4},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			defer logger.SetLogLevelWithRestore(tc.level)()
			for i, check := range allChecks {
				require.Equal(i <= tc.activeIdx, check())
			}
		})
	}
}

func Test_WithContextAttrs(t *testing.T) {
	t.Run("attrs appear in output", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		ctx := logger.WithContextAttrs(context.Background(), map[string]any{
			logger.LogAttr_VApp: "untill.fiscalcloud",
			logger.LogAttr_Feat: "magicmenu",
		})
		logger.VerboseCtx(ctx, "", "hello ctx")
		logCap.HasLine("hello ctx", "vapp=untill.fiscalcloud", "feat=magicmenu")
	})

	t.Run("attrs accumulate across WithContextAttrs calls", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		ctx := logger.WithContextAttrs(context.Background(), map[string]any{logger.LogAttr_VApp: "myapp"})
		ctx = logger.WithContextAttrs(ctx, map[string]any{logger.LogAttr_Feat: "myfeat"})
		logger.VerboseCtx(ctx, "", "accumulated")
		logCap.HasLine("vapp=myapp", "feat=myfeat")
	})

	t.Run("same key is overwritten", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		ctx := logger.WithContextAttrs(context.Background(), map[string]any{logger.LogAttr_VApp: "first"})
		ctx = logger.WithContextAttrs(ctx, map[string]any{logger.LogAttr_VApp: "second"})
		logger.VerboseCtx(ctx, "", "overwrite")
		logCap.HasLine("vapp=second")
		logCap.NotContains("vapp=first")
	})
}

func Test_CtxFuncs_StandardAttrs(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelTrace)
	ctx := logger.WithContextAttrs(context.Background(), map[string]any{
		logger.LogAttr_ReqID:     42,
		logger.LogAttr_WSID:      100,
		logger.LogAttr_Extension: "c.sys.UploadBLOBHelper",
	})
	logger.InfoCtx(ctx, "", "standard attrs")
	logCap.HasLine("reqid=42", "wsid=100", "extension=c.sys.UploadBLOBHelper")
}

func Test_CtxFuncs_SLogLevels(t *testing.T) {
	testCases := []struct {
		name      string
		level     logger.TLogLevel
		logFn     func(context.Context, string, ...interface{})
		msg       string
		wantLevel string
	}{
		{name: "ErrorCtx", level: logger.LogLevelError, logFn: logger.ErrorCtx, msg: "error msg", wantLevel: "ERROR"},
		{name: "WarningCtx", level: logger.LogLevelWarning, logFn: logger.WarningCtx, msg: "warning msg", wantLevel: "WARN"},
		{name: "InfoCtx", level: logger.LogLevelInfo, logFn: logger.InfoCtx, msg: "info msg", wantLevel: "INFO"},
		{name: "VerboseCtx", level: logger.LogLevelVerbose, logFn: logger.VerboseCtx, msg: "verbose msg", wantLevel: "DEBUG"},
		{name: "TraceCtx", level: logger.LogLevelTrace, logFn: logger.TraceCtx, msg: "trace msg", wantLevel: "DEBUG"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logCap := logger.StartCapture(t, tc.level)
			tc.logFn(context.Background(), "", tc.msg)
			logCap.HasLine("level="+tc.wantLevel, "msg=\""+tc.msg+"\"")
		})
	}
}

func Test_CtxFuncs_LevelFiltering(t *testing.T) {
	ctx := context.Background()

	t.Run("VerboseCtx suppressed at Info level", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelInfo)
		logger.VerboseCtx(ctx, "", "should not appear")
		logCap.NotContains("should not appear")
	})

	t.Run("TraceCtx suppressed at Info level", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelInfo)
		logger.TraceCtx(ctx, "", "should not appear")
		logCap.NotContains("should not appear")
	})

	t.Run("ErrorCtx captured with attrs", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelError)
		ctx2 := logger.WithContextAttrs(ctx, map[string]any{"k": "v"})
		logger.ErrorCtx(ctx2, "", "boom")
		logCap.HasLine("boom", "k=v")
	})

	t.Run("WarningCtx visible at Warning level", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelWarning)
		logger.WarningCtx(ctx, "", "warn msg")
		logCap.HasLine("warn msg")
	})
}

func Test_CtxFuncs_EmptyContext(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	logger.VerboseCtx(context.Background(), "", "no attrs")
	logCap.HasLine("no attrs")
}

func Test_CtxFuncs_StageAttr(t *testing.T) {
	t.Run("stage appears when non-empty", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		logger.VerboseCtx(context.Background(), "endpoint.validation", "test msg")
		logCap.HasLine("stage=endpoint.validation", "test msg")
	})

	t.Run("stage omitted when empty", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		logger.VerboseCtx(context.Background(), "", "no stage msg")
		logCap.HasLine("no stage msg")
		logCap.NotContains("stage=")
	})
}

func TestMultithread(t *testing.T) {
	toLog := []string{}
	for i := 0; i < 100; i++ {
		toLog = append(toLog, strings.Repeat(strconv.Itoa(i), 10))
	}

	logCap := logger.StartCapture(t, logger.LogLevelInfo)
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			for i := 0; i < 100; i++ {
				logger.Info(toLog[i])
			}
			wg.Done()
		}()
	}
	wg.Wait()

	logged := strings.Split(logCap.String(), "\n")
outer:
	for _, loggedActual := range logged {
		if len(loggedActual) == 0 {
			continue
		}
		for _, loggedExpected := range toLog {
			if strings.Contains(loggedActual, loggedExpected) {
				continue outer
			}
		}
		t.Fatal(loggedActual)
	}
}
