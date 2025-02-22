/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

func NewElectionsTTLStorage(vs IVVMAppTTLStorage) ITTLStorage[TTLStorageImplKey, string] {
	return &implITTLStorageElections{
		prefix:        pKeyPrefix_Elections,
		vvmttlstorage: vs,
	}
}

