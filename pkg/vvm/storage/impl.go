/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */
package storage

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

type implStorageBase struct {
	prefix        pKeyPrefix
	sysVVMStorage ISysVvmStorage
}

func (s *implStorageBase) getPKey() (pKey []byte) {
	pKey = make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(pKey, s.prefix)
	return pKey
}
