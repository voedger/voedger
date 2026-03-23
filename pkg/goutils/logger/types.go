/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"bytes"
	"sync"
)

type ctxKey struct{}

type logAttrs struct {
	attrs  map[string]any
	parent *logAttrs
}

// copy of testing.TB to avoid importing testing package in production
type TB interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...interface{})
	FailNow()
}

type ILogCaptor interface {
	String() string
	HasLine(str string, strs ...string)           // fails if no single line contains all substrings (any order)
	EventuallyHasLine(str string, strs ...string) // same, retries up to 1 second
	NotContains(str string, strs ...string)       // fails if any substring appears in the log
	Reset()
}

type captor struct {
	mu  sync.Mutex
	buf bytes.Buffer
	t   TB
}
