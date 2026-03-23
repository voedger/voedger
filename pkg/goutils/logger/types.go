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

type TB interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...interface{})
	FailNow()
}

type ILogCaptor interface {
	String() string
	HasLine(strs ...string)
	EventuallyHasLine(strs ...string) // waits for 1 second
	NotContains(strs ...string)
	Reset()
}
