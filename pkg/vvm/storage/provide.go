/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

func NewElectionsTTLStorage(vs ISysVvmStorage) ITTLStorage[TTLStorageImplKey, string] {
	return &implElectionsITTLStorage{
		prefix:        pKeyPrefix_VVMLeader,
		vvmttlstorage: vs,
	}
}
