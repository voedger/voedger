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

// ReleaseWLogEvent releases the owned reference when a state reader discards a WLog value.
func ReleaseWLogEvent(value istructs.IStateValue) {
	if wlog, ok := value.(istructs.IStateWLogValue); ok {
		wlog.AsEvent().Release()
	}
}

func ReadSecret(state istructs.IState, name string) (value string, err error) {
	kb, err := state.KeyBuilder(sys.Storage_AppSecret, sys.Storage_AppSecret)
	if err != nil {
		// notest
		return "", err
	}
	kb.PutString(sys.Storage_AppSecretField_Secret, name)
	sv, err := state.MustExist(kb)
	if err != nil {
		return "", err
	}
	return sv.AsString(""), nil
}
