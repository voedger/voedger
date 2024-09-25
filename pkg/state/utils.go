/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package state

import (
	"github.com/voedger/voedger/pkg/istructs"
)

func SimpleWSIDFunc(wsid istructs.WSID) WSIDFunc {
	return func() istructs.WSID { return wsid }
}
func SimplePartitionIDFunc(partitionID istructs.PartitionID) PartitionIDFunc {
	return func() istructs.PartitionID { return partitionID }
}
