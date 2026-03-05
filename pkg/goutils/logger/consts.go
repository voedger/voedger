/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"log/slog"
	"os"
)

const (
	logCtxSkipFrames = 3
	slogLevelTrace   = 4
)

const (
	LogAttr_VApp      = "vapp"
	LogAttr_Feat      = "feat"
	LogAttr_ReqID     = "reqid"
	LogAttr_WSID      = "wsid"
	LogAttr_Extension = "extension"
)

var (
	// ctxHandlerOpts disables handler-level filtering (isEnabled() already gates all calls)
	// and maps internal slog levels to the names used by the logger package.
	ctxHandlerOpts = &slog.HandlerOptions{
		Level: slog.LevelDebug - slogLevelTrace,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				switch a.Value.Any().(slog.Level) {
				case slog.LevelDebug:
					a.Value = slog.StringValue("VERBOSE")
				case slog.LevelDebug - slogLevelTrace:
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	}
	slogOut = slog.New(slog.NewTextHandler(os.Stdout, ctxHandlerOpts))
	slogErr = slog.New(slog.NewTextHandler(os.Stderr, ctxHandlerOpts))
)
