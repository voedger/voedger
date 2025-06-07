/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/istructs"
)

func CrackID(id istructs.RecordID) int64 {
	return sys.RecordsRegistryView.Fields.CrackID(id)
}
