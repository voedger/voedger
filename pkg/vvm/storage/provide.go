/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import "github.com/voedger/voedger/pkg/elections"

func NewElectionsTTLStorage(vs ISysVvmStorage) elections.ITTLStorage[TTLStorageImplKey, string] {
	return &implElectionsITTLStorage{
		prefix:        pKeyPrefix_VVMLeader,
		vvmttlstorage: vs,
	}
}
