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
	logCap := StartCapture(t, LogLevelVerbose)
	t.Run("captures InfoCtx output", func(t *testing.T) {
		require := require.New(t)
		InfoCtx(context.Background(), "captured message")
		require.Contains(logCap.String(), "captured message")
	})
	t.Run("captures context attrs", func(t *testing.T) {
		require := require.New(t)
		ctx := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 42, LogAttr_VApp: "myapp"})
		InfoCtx(ctx, "with attrs")
		require.Contains(logCap.String(), "wsid=42")
		require.Contains(logCap.String(), "vapp=myapp")
	})
	t.Run("multiple strings on same line", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		ctx := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 99})
		InfoCtx(ctx, "my message")
		logCap.HasLine("wsid=99", "my message")
	})
	t.Run("ErrorCtx is captured", func(t *testing.T) {
		require := require.New(t)
		ErrorCtx(context.Background(), "error message")
		require.Contains(logCap.String(), "error message")
	})
	t.Run("does not capture below set level", func(t *testing.T) {
		require := require.New(t)
		capInfo := StartCapture(t, LogLevelInfo)
		VerboseCtx(context.Background(), "verbose msg")
		require.Empty(capInfo.String())
	})
	t.Run("HasLine matches per-line only, no cross-line interference", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		ctx1 := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 1})
		ctx2 := WithContextAttrs(context.Background(), map[string]any{LogAttr_WSID: 2})
		InfoCtx(ctx1, "alpha")
		InfoCtx(ctx2, "beta")

		logCap.HasLine("alpha", "wsid=1")
		logCap.HasLine("beta", "wsid=2")
	})

	t.Run("restores writers after test cleanup", func(t *testing.T) {
		var innerCap ILogCaptor
		t.Run("inner", func(t *testing.T) {
			innerCap = StartCapture(t, LogLevelVerbose)
			InfoCtx(context.Background(), "inner msg")
		})
		// inner t.Cleanup has fired by now; slogOut is restored to the pre-inner state
		InfoCtx(context.Background(), "post-cleanup msg")
		require.NotContains(t, innerCap.String(), "post-cleanup msg")
	})
}

func TestHasNoLines(t *testing.T) {
	t.Run("absent string passes", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		InfoCtx(context.Background(), "hello")
		logCap.HasNoLines("notpresent")
	})
	t.Run("empty buffer passes", func(t *testing.T) {
		logCap := StartCapture(t, LogLevelVerbose)
		logCap.HasNoLines("anything")
	})
}

func TestReset(t *testing.T) {
	require := require.New(t)
	locCap := StartCapture(t, LogLevelVerbose)
	InfoCtx(context.Background(), "before reset")
	locCap.Reset()
	require.Empty(locCap.String())
	InfoCtx(context.Background(), "after reset")
	locCap.HasLine("after reset")
	locCap.HasNoLines("before reset")
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
}

func TestEventuallyHasLine(t *testing.T) {
	logCap := StartCapture(t, LogLevelVerbose)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		InfoCtx(context.Background(), "async message")
	}()
	logCap.EventuallyHasLine("async message")
	wg.Wait()
}
