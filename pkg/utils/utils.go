/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"
	"os"
	"strings"
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

func ServerAddress(port int) string {
	addr := ""
	if IsTest() {
		addr = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", addr, port)
}
