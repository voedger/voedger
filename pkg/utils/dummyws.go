/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"sync"

	"github.com/voedger/voedger/pkg/istructs"
)

var dummyWSLock = sync.RWMutex{}
var dummyWSes = map[istructs.WSID]bool{}

func IsDummyWS(wsid istructs.WSID) (res bool) {
	if !IsTest() {
		return false
	}
	dummyWSLock.RLock()
	res = dummyWSes[wsid]
	dummyWSLock.RUnlock()
	return res
}

// command processor will skip initialization check for the wsid
// must be used in tests only
func AddDummyWS(wsid istructs.WSID) {
	dummyWSLock.Lock()
	dummyWSes[wsid] = true
	dummyWSLock.Unlock()
}
