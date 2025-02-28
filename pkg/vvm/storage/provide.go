/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import "github.com/voedger/voedger/pkg/ielections"

// [~server.design.orch/NewElectionsTTLStorage~impl]
func NewElectionsTTLStorage(vs ISysVvmStorage) ielections.ITTLStorage[TTLStorageImplKey, string] {
	return &implElectionsITTLStorage{
		prefix:        pKeyPrefix_VVMLeader,
		vvmttlstorage: vs,
	}
}
