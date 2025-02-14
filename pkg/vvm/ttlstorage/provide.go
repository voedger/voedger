/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ttlstorage

import (
	"github.com/voedger/voedger/pkg/elections"
)

// New constructs an `elections.ITTLStorage[TTLStorageImplKey, string]` with the given key prefix
// and IVVMAppTTLStorage implementation.
func New(keyPrefix PKeyPrefix, vs IVVMAppTTLStorage) elections.ITTLStorage[TTLStorageImplKey, string] {
	return &storageImpl{
		prefix:        keyPrefix,
		vvmttlstorage: vs,
	}
}
