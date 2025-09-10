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
	"testing"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
)

func IsBlank(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}

func IsTest() bool {
	return testing.Testing() || IsDebug()
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

func ListenAddr(port int) string {
	if port == 0 {
		return LocalhostDynamic()
	}
	return fmt.Sprintf(":%d", port)
}

// LocalhostDynamic returns a server address that binds only to localhost (127.0.0.1:0)
// Use this for tests, admin interfaces, or services that should only be locally accessible
func LocalhostDynamic() string {
	return fmt.Sprintf("%s:0", LocalhostIP.String())
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

func Int64ToWSID(val int64) (istructs.WSID, error) {
	if val < 0 || val > istructs.MaxAllowedWSID {
		return 0, errors.New("wsid value is out of range:" + utils.IntToString(val))
	}
	return istructs.WSID(val), nil
}

func Int64ToRecordID(val int64) (istructs.RecordID, error) {
	if val < 0 {
		return 0, errors.New("record ID value is out of range:" + utils.IntToString(val))
	}
	return istructs.RecordID(val), nil
}
