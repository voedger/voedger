/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"github.com/voedger/voedger/pkg/elections"
	"github.com/voedger/voedger/pkg/vvm"
)

func NewElectionsTTLStorage(vs vvm.IVVMAppTTLStorage) elections.ITTLStorage[TTLStorageImplKey, string] {
	return &implITTLStorageElections{
		prefix:        pKeyPrefix_Elections,
		vvmttlstorage: vs,
	}
}
