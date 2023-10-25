/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import "github.com/voedger/voedger/pkg/istructs"

func crackID(id istructs.RecordID) uint64 {
	return uint64(id >> registryViewBits)
}
