/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package state

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func SimpleWSIDFunc(wsid istructs.WSID) WSIDFunc {
	return func() istructs.WSID { return wsid }
}

func SimplePartitionIDFunc(partitionID istructs.PartitionID) PartitionIDFunc {
	return func() istructs.PartitionID { return partitionID }
}

func ReadSecret(state istructs.IState, name string) (value string, err error) {
	kb, err := state.KeyBuilder(sys.Storage_AppSecret, sys.Storage_AppSecret)
	if err != nil {
		// notest
		return
	}
	kb.PutString(sys.Storage_AppSecretField_Secret, name)
	sv, err := state.MustExist(kb)
	if err != nil {
		return
	}
	return sv.AsString(""), nil
}
