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

	"github.com/voedger/voedger/pkg/istructs"
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

func PartitionID(wsid istructs.WSID, numCommandProcessors CommandProcessorsCount) istructs.PartitionID {
	return istructs.PartitionID(int(wsid) % int(numCommandProcessors))
}

func SplitErrors(joinedError error) (errs []error) {
	if joinedError != nil {
		if e, ok := joinedError.(interface{ Unwrap() []error }); ok {
			return e.Unwrap()
		}
		return []error{joinedError}
	}
	return
}
