/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"encoding/binary"
	"time"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/vvm"
)

type implITTLStorageElections struct {
	prefix        pKeyPrefix
	vvmttlstorage vvm.IVVMAppTTLStorage
}

func (s *implITTLStorageElections) buildKeys(key TTLStorageImplKey) (pKey, cCols []byte) {
	pKey = make([]byte, utils.Uint32Size)
	cCols = make([]byte, utils.Uint32Size)

	binary.BigEndian.PutUint32(pKey, s.prefix)
	binary.BigEndian.PutUint32(cCols, key)
	return
}

func (s *implITTLStorageElections) InsertIfNotExist(key TTLStorageImplKey, val string, ttl time.Duration) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.vvmttlstorage.InsertIfNotExists(pKey, cCols, []byte(val), int(ttl.Seconds()))
}

func (s *implITTLStorageElections) CompareAndSwap(key TTLStorageImplKey, oldVal, newVal string, ttl time.Duration) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.vvmttlstorage.CompareAndSwap(pKey, cCols, []byte(oldVal), []byte(newVal), int(ttl.Seconds()))
}

func (s *implITTLStorageElections) CompareAndDelete(key TTLStorageImplKey, val string) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.vvmttlstorage.CompareAndDelete(pKey, cCols, []byte(val))
}
