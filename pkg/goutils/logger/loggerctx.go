/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// SetCtxWriters replaces the writers used by all *Ctx logging functions.
// Must be used in tests only: create an os.Pipe(), call SetCtxWriters, run the code, read the pipe.
func SetCtxWriters(out, err io.Writer) {
	slogOut = slog.New(slog.NewTextHandler(out, ctxHandlerOpts))
	slogErr = slog.New(slog.NewTextHandler(err, ctxHandlerOpts))
}

// WithContextAttrs returns a new context with the given slog attributes
// added to any already stored. Attributes with the same key are overwritten.
// Thread-safe: uses a fresh sync.Map per call so callers do not share mutable state.
func WithContextAttrs(ctx context.Context, name string, value any) context.Context {
	prev, _ := ctx.Value(ctxKey{}).(*sync.Map)
	m := &sync.Map{}
	if prev != nil {
		prev.Range(func(k, v any) bool {
			m.Store(k, v)
			return true
		})
	}
	m.Store(name, value)
	return context.WithValue(ctx, ctxKey{}, m)
}

// Context-aware logging functions. Each reads slog.Attr values stored in ctx
// (via WithContextAttrs) and appends them to the log record.

func VerboseCtx(ctx context.Context, args ...interface{}) {
	logCtx(ctx, LogLevelVerbose, slog.LevelDebug, args...)
}

func ErrorCtx(ctx context.Context, args ...interface{}) {
	logCtx(ctx, LogLevelError, slog.LevelError, args...)
}

func InfoCtx(ctx context.Context, args ...interface{}) {
	logCtx(ctx, LogLevelInfo, slog.LevelInfo, args...)
}

func WarningCtx(ctx context.Context, args ...interface{}) {
	logCtx(ctx, LogLevelWarning, slog.LevelWarn, args...)
}

func TraceCtx(ctx context.Context, args ...interface{}) {
	logCtx(ctx, LogLevelTrace, slog.LevelDebug-4, args...)
}

// logCtx is the shared implementation for all *Ctx functions.
func logCtx(ctx context.Context, level TLogLevel, slogLevel slog.Level, args ...interface{}) {
	if !isEnabled(level) {
		return
	}
	log := slogOut
	if level == LogLevelError {
		log = slogErr
	}
	attrsFromCtx := sLogAttrsFromCtx(ctx)
	attrs := make([]any, 0, len(attrsFromCtx)+1)
	fn, line := getFuncName(logCtxSkipFrames)
	attrs = append(attrs, slog.Attr{Key: "src", Value: slog.StringValue(fmt.Sprintf("%s:%d", fn, line))})
	attrs = append(attrs, attrsFromCtx...)
	log.Log(ctx, slogLevel, fmt.Sprint(args...), attrs...)
}

func sLogAttrsFromCtx(ctx context.Context) []any {
	m, _ := ctx.Value(ctxKey{}).(*sync.Map)
	if m == nil {
		return nil
	}
	var attrs []any
	m.Range(func(k, v any) bool {
		attrs = append(attrs, slog.Any(k.(string), v))
		return true
	})
	return attrs
}
