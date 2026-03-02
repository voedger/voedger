/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

type ctxKey struct{}

type logAttrs struct {
	attrs  map[string]any
	parent *logAttrs
}
