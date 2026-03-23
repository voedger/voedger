/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package logger

import (
	"bytes"
	"sync"
	"time"
)

type captor struct {
	mu  sync.Mutex
	buf bytes.Buffer
	t   TB
}

func StartCapture(t TB, level TLogLevel) ILogCaptor {
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
		c.t.Errorf("no log line contains all of the expected substrings\nlog:\n%s", content)
		c.t.FailNow()
	}
}

func (c *captor) EventuallyHasLine(strs ...string) {
	c.t.Helper()
	deadline := time.Now().Add(eventuallyHasLineTimeout)
	for {
		c.mu.Lock()
		content := bytes.Clone(c.buf.Bytes())
		c.mu.Unlock()
		if anyLineContainsAll(content, strs) {
			return
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.mu.Lock()
	content := bytes.Clone(c.buf.Bytes())
	c.mu.Unlock()
	c.t.Errorf("no log line contains all of the expected substrings\nexpected: %v\nlog:\n%s", strs, content)
	c.t.FailNow()
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

func (c *captor) NotContains(strs ...string) {
	c.t.Helper()
	c.mu.Lock()
	content := bytes.Clone(c.buf.Bytes())
	c.mu.Unlock()
	for _, s := range strs {
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
