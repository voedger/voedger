/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"sync"

	"github.com/untillpro/voedger/pkg/istructs"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

var dummyWSLock = sync.RWMutex{}
var dummyWSes = map[istructs.WSID]bool{}

func IsDummyWS(wsid istructs.WSID) (res bool) {
	if !coreutils.IsTest() {
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
