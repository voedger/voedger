/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"github.com/voedger/voedger/pkg/ielections"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
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

func NewVVMSeqStorageAdapter(vs ISysVvmStorage, partitionID istructs.PartitionID) isequencer.IVVMSeqStorageAdapter {
	return &implVVMSeqStorageAdapter{
		
	}
}
