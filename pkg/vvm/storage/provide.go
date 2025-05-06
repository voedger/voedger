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
func NewElectionsTTLStorage(sysVVMStorage ISysVvmStorage) ielections.ITTLStorage[TTLStorageImplKey, string] {
	return &implIElectionsTTLStorage{
		sysVVMStorage: sysVVMStorage,
	}
}

func NewVVMSeqStorageAdapter(sysVVMStorage ISysVvmStorage, partitionID isequencer.PartitionID) isequencer.IVVMSeqStorageAdapter {
	return &implVVMSeqStorageAdapter{
		partitionID:   partitionID,
		sysVVMStorage: sysVVMStorage,
	}
}
