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
)

// SetCtxWriters replaces the writers used by all *Ctx logging functions.
// Must be used in tests only: create an os.Pipe(), call SetCtxWriters, run the code, read the pipe.
func SetCtxWriters(out, err io.Writer) {
	slogOut = slog.New(slog.NewTextHandler(out, ctxHandlerOpts))
	slogErr = slog.New(slog.NewTextHandler(err, ctxHandlerOpts))
}

// WithContextAttrs returns a new context with the given attributes added. Later calls shadow earlier ones for the same key.
func WithContextAttrs(ctx context.Context, attrs map[string]any) context.Context {
	prev, _ := ctx.Value(ctxKey{}).(*logAttrs)
	return context.WithValue(ctx, ctxKey{}, &logAttrs{attrs: attrs, parent: prev})
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
	node, _ := ctx.Value(ctxKey{}).(*logAttrs)
	seen := map[string]bool{}
	attrs := []any{}
	for node != nil {
		for k := range node.attrs {
			if seen[k] {
				continue
			}
			seen[k] = true
			attrs = append(attrs, slog.Any(k, node.attrs[k]))
		}
		node = node.parent
	}
	return attrs
}
