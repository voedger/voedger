/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func StartCapture(t testing.TB, level TLogLevel) ILogCaptor {
	captor := &captor{
		t: t,
	}
	oldSLogOut, oldSLogErr := slogOut, slogErr
	SetCtxWriters(captor, captor)
	restoreLevel := SetLogLevelWithRestore(level)
	t.Cleanup(func() {
		slogOut, slogErr = oldSLogOut, oldSLogErr
		restoreLevel()
	})
	return captor
}

func (c *captor) HasLine(strs ...string) {
	c.t.Helper()
	c.mu.Lock()
	content := bytes.Clone(c.buf.Bytes())
	c.mu.Unlock()
	if !anyLineContainsAll(content, strs) {
		require.Fail(c.t, "no log line contains all of the expected substrings",
			"log:\n%s", content)
	}
}

func (c *captor) EventuallyHasLine(strs ...string) {
	c.t.Helper()
	ok := assert.Eventually(c.t, func() bool {
		c.mu.Lock()
		content := bytes.Clone(c.buf.Bytes())
		c.mu.Unlock()
		return anyLineContainsAll(content, strs)
	}, eventuallyHasLineTimeout, 10*time.Millisecond)
	if !ok {
		c.mu.Lock()
		content := bytes.Clone(c.buf.Bytes())
		c.mu.Unlock()
		require.Fail(c.t, "no log line contains all of the expected substrings",
			"expected: %v\nlog:\n%s", strs, content)
	}
}

func anyLineContainsAll(content []byte, strs []string) bool {
	for _, line := range bytes.Split(content, []byte("\n")) {
		found := true
		for _, s := range strs {
			if !bytes.Contains(line, []byte(s)) {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

func (c *captor) HasNoLines(strs ...string) {
	c.t.Helper()
	c.mu.Lock()
	content := bytes.Clone(c.buf.Bytes())
	c.mu.Unlock()
	for _, s := range strs {
		require.NotContains(c.t, content, []byte(s), "log:\n%s", content)
	}
}

func (c *captor) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func (c *captor) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *captor) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buf.Reset()
}
