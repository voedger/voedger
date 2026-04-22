/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogCapture_BasicUsage(t *testing.T) {
	require := require.New(t)
	t.Run("captures InfoCtx output", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		InfoCtx(context.Background(), "test.stage", "captured message")
		require.Contains(logCap.String(), "captured message")
	})
	t.Run("captures context attrs", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		ctx := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 42, LogAttr_VApp: "myapp"})
		InfoCtx(ctx, "test.stage", "with attrs")
		require.Contains(logCap.String(), "wsid=42")
		require.Contains(logCap.String(), "vapp=myapp")
	})
	t.Run("multiple strings on same line", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		ctx := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 99})
		InfoCtx(ctx, "test.stage", "my message")
		logCap.HasLine("wsid=99", "my message")
	})
	t.Run("ErrorCtx is captured", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		ErrorCtx(context.Background(), "test.stage", "error message")
		require.Contains(logCap.String(), "error message")
	})
	t.Run("does not capture below set level", func(t *testing.T) {
		capInfo := StartCapture(t, LogLevelInfo)
		VerboseCtx(context.Background(), "test.stage", "verbose msg")
		require.Empty(capInfo.String())
	})
	t.Run("HasLine matches per-line only, no cross-line interference", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		ctx1 := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 1})
		ctx2 := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 2})
		InfoCtx(ctx1, "test.stage", "alpha")
		InfoCtx(ctx2, "test.stage", "beta")

		logCap.HasLine("alpha", "wsid=1")
		logCap.HasLine("beta", "wsid=2")
	})

	t.Run("restores writers after test cleanup", func(t *testing.T) {
		var innerCap ILogCaptor
		t.Run("inner", func(t *testing.T) {
			innerCap = StartCapture(t, LogLevelVerbose)
			InfoCtx(context.Background(), "test.stage", "inner msg")
		})
		// inner t.Cleanup has fired by now; slogOut is restored to the pre-inner state
		InfoCtx(context.Background(), "test.stage", "post-cleanup msg")
		require.NotContains(innerCap.String(), "post-cleanup msg")
	})
}

func TestNotContains(t *testing.T) {
	t.Run("absent string passes", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		InfoCtx(context.Background(), "test.stage", "hello")
		logCap.NotContains("notpresent")
	})
	t.Run("empty buffer passes", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		logCap.NotContains("anything")
	})
}

func TestReset(t *testing.T) {
	require := require.New(t)
	locCap := StartCapture(t, LogLevelVerbose)
	InfoCtx(context.Background(), "test.stage", "before reset")
	locCap.Reset()
	require.Empty(locCap.String())
	InfoCtx(context.Background(), "test.stage", "after reset")
	locCap.HasLine("after reset")
	locCap.NotContains("before reset")
}

func TestCaptor_Failure(t *testing.T) {
	runFailure := func(t *testing.T, fn func(*captor)) {
		t.Helper()
		require := require.New(t)
		spy := &testing.T{}
		c := &captor{t: spy}
		done := make(chan struct{})
		go func() {
			defer close(done)
			fn(c)
		}()
		<-done
		require.True(spy.Failed())
	}
	t.Run("HasLine", func(t *testing.T) {
		runFailure(t, func(c *captor) { c.HasLine("absent string") })
	})
	t.Run("EventuallyHasLine", func(t *testing.T) {
		runFailure(t, func(c *captor) { c.EventuallyHasLine("absent string") })
	})
	t.Run("NotContains", func(t *testing.T) {
		runFailure(t, func(c *captor) {
			c.buf.WriteString("present string\n")
			c.NotContains("present string")
		})
	})
}

func TestEventuallyHasLine(t *testing.T) {
	logCap := StartCapture(t, LogLevelVerbose)
	var wg sync.WaitGroup
	wg.Go(func() {
		time.Sleep(100 * time.Millisecond)
		InfoCtx(context.Background(), "test.stage", "async message")
	})
	logCap.EventuallyHasLine("async message")
	wg.Wait()
}

func TestLegacyFunctions(t *testing.T) {
	testCases := []struct {
		name         string
		captureLevel TLogLevel
		logFn        func(...interface{})
		wantLine     []string // wantLine[0] is logged; HasLine(wantLine[0], wantLine[1:]...) is asserted
		notContains  bool     // true → NotContains(wantLine[0]) instead of HasLine
	}{
		{name: "Verbose captured", captureLevel: LogLevelVerbose, logFn: Verbose, wantLine: []string{"verbose msg"}},
		{name: "Info captured", captureLevel: LogLevelInfo, logFn: Info, wantLine: []string{"info msg"}},
		{name: "Warning captured", captureLevel: LogLevelWarning, logFn: Warning, wantLine: []string{"warning msg"}},
		{name: "Error captured", captureLevel: LogLevelError, logFn: Error, wantLine: []string{"error msg"}},
		{name: "Trace captured", captureLevel: LogLevelTrace, logFn: Trace, wantLine: []string{"trace msg"}},
		{name: "Verbose not captured at Info level", captureLevel: LogLevelInfo, logFn: Verbose, wantLine: []string{"should not appear"}, notContains: true},
		{name: "Error has error prefix", captureLevel: LogLevelError, logFn: Error, wantLine: []string{"my error", errorPrefix}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logCap := StartCapture(t, tc.captureLevel)
			tc.logFn(tc.wantLine[0])
			if tc.notContains {
				logCap.NotContains(tc.wantLine[0])
			} else {
				logCap.HasLine(tc.wantLine[0], tc.wantLine[1:]...)
			}
		})
	}
}
