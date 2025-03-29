/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"github.com/voedger/voedger/pkg/ielections"
	"github.com/voedger/voedger/pkg/isequencer"
)

// [~server.design.orch/NewElectionsTTLStorage~impl]
func NewElectionsTTLStorage(vs ISysVvmStorage) ielections.ITTLStorage[TTLStorageImplKey, string] {
	return &implIElectionsTTLStorage{
		implStorageBase: implStorageBase{
			prefix:        pKeyPrefix_VVMLeader,
			sysVVMStorage: vs,
		},
	}
}

func NewSeqStorage(vs ISysVvmStorage) isequencer.ISeqSysVVMStorage {
	return &implISeqSysVVMStorage{
		implStorageBase: implStorageBase{
			prefix:        pKeyPrefix_SeqStorage,
			sysVVMStorage: vs,
		},
	}
}
