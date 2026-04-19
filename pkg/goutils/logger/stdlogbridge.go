/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"bytes"
	"context"
	"log"
)

func WithFilter(substrings ...string) StdLogBridgeOption {
	return func(w *stdLogBridgeWriter) {
		for _, str := range substrings {
			if len(str) == 0 {
				continue
			}
			w.filters = append(w.filters, []byte(str))
		}
	}
}

func NewStdErrorLogBridge(ctx context.Context, stage string, opts ...StdLogBridgeOption) *log.Logger {
	w := &stdLogBridgeWriter{ctx: ctx, stage: stage, logLevel: LogLevelError}
	for _, opt := range opts {
		opt(w)
	}
	return log.New(w, "", 0)
}

func (w *stdLogBridgeWriter) Write(p []byte) (int, error) {
	n := len(p)
	if !isEnabled(w.logLevel) {
		return n, nil
	}
	for _, s := range w.filters {
		if bytes.Contains(p, s) {
			return n, nil
		}
	}
	trimmed := bytes.TrimRight(p, "\r\n")
	if len(trimmed) == 0 {
		return n, nil
	}
	LogCtx(w.ctx, stdLogBridgeSkipStackFrames, w.logLevel, w.stage, string(trimmed))
	return n, nil
}
