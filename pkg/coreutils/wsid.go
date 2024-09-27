/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
)

func Int64ToWSID(val int64) (istructs.WSID, error) {
	if val < 0 || val > istructs.MaxAllowedWSID {
		panic("wsid value is out of range:" + utils.IntToString(val))
	}
	return istructs.WSID(val), nil
}

func Int64ToRecordID(val int64) (istructs.RecordID, error) {
	if val < 0 {
		panic("record ID value is out of range:" + utils.IntToString(val))
	}
	return istructs.RecordID(val), nil
}
