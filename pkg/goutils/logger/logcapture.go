/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"bytes"
	"time"
)

func StartCapture(t TB, level TLogLevel) ILogCaptor {
	c := &captor{
		t: t,
	}
	oldSLogOut, oldSLogErr := slogOut, slogErr
	oldLegacyOut, oldLegacyErr := legacyOut, legacyErr
	SetCtxWriters(c, c)
	legacyOut, legacyErr = c, c
	restoreLevel := SetLogLevelWithRestore(level)
	t.Cleanup(func() {
		slogOut, slogErr = oldSLogOut, oldSLogErr
		legacyOut, legacyErr = oldLegacyOut, oldLegacyErr
		restoreLevel()
	})
	return c
}

func (c *captor) HasLine(str string, strs ...string) {
	c.t.Helper()
	c.mu.Lock()
	content := bytes.Clone(c.buf.Bytes())
	c.mu.Unlock()
	if !anyLineContainsAll(content, str, strs) {
		c.t.Errorf("no log line contains all of the expected substrings\nlog:\n%s", content)
		c.t.FailNow()
	}
}

func (c *captor) EventuallyHasLine(str string, strs ...string) {
	c.t.Helper()
	deadline := time.Now().Add(eventuallyHasLineTimeout)
	var content []byte
	for {
		c.mu.Lock()
		content = bytes.Clone(c.buf.Bytes())
		c.mu.Unlock()
		if anyLineContainsAll(content, str, strs) {
			return
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.t.Errorf("no log line contains all of the expected substrings\nexpected: %v\nlog:\n%s", append([]string{str}, strs...), content)
	c.t.FailNow()
}

func anyLineContainsAll(content []byte, str string, strs []string) bool {
	all := append([]string{str}, strs...)
	for _, line := range bytes.Split(content, []byte("\n")) {
		found := true
		for _, s := range all {
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

func (c *captor) NotContains(str string, strs ...string) {
	c.t.Helper()
	c.mu.Lock()
	content := bytes.Clone(c.buf.Bytes())
	c.mu.Unlock()
	for _, s := range append([]string{str}, strs...) {
		if bytes.Contains(content, []byte(s)) {
			c.t.Errorf("log contains unexpected string %q\nlog:\n%s", s, content)
			c.t.FailNow()
		}
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
