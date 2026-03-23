/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"bytes"
	"sync"
	"testing"
)

type ctxKey struct{}

type logAttrs struct {
	attrs  map[string]any
	parent *logAttrs
}

type captor struct {
	mu  sync.Mutex
	buf bytes.Buffer
	t   testing.TB
}

type ILogCaptor interface {
	String() string
	HasLine(strs ...string)
	EventuallyHasLine(strs ...string) // waits for 1 second
	HasNoLines(strs ...string)
	Reset()
}
