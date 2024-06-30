/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

func IsBlank(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}

// https://github.com/golang/go/issues/27169
func ResetTimer(t *time.Timer, timeout time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(timeout)
}
func IsTest() bool {
	return strings.Contains(os.Args[0], ".test") || IsDebug()
}

func IsDebug() bool {
	return strings.Contains(os.Args[0], "__debug_bin")
}

func IsCassandraStorage() bool {
	_, ok := os.LookupEnv("CASSANDRA_TESTS_ENABLED")
	return ok
}

func IsDynamoDBStorage() bool {
	_, ok := os.LookupEnv("DYNAMODB_TESTS_ENABLED")
	return ok
}

func ServerAddress(port int) string {
	addr := ""
	if IsTest() {
		addr = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", addr, port)
}

func SplitErrors(joinedError error) (errs []error) {
	if joinedError != nil {
		var pErr IErrUnwrapper
		if errors.As(joinedError, &pErr) {
			return pErr.Unwrap()
		}
		return []error{joinedError}
	}
	return
}

func IsWSAEError(err error, errno syscall.Errno) bool {
	var sysCallErr *os.SyscallError
	if errors.As(err, &sysCallErr) {
		var syscallErrno syscall.Errno
		if errors.As(sysCallErr.Err, &syscallErrno) {
			return syscallErrno == errno
		}
	}
	return false
}

func NilAdminPortGetter() int { panic("to be tested") }

func ScanSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
