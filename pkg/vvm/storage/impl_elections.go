/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

type implElectionsITTLStorage struct {
	prefix        pKeyPrefix
	vvmttlstorage ISysVvmStorage
}

func (s *implElectionsITTLStorage) buildKeys(key TTLStorageImplKey) (pKey, cCols []byte) {
	pKey = make([]byte, utils.Uint32Size)
	cCols = make([]byte, utils.Uint32Size)

	binary.BigEndian.PutUint32(pKey, s.prefix)
	binary.BigEndian.PutUint32(cCols, key)
	return
}

func (s *implElectionsITTLStorage) InsertIfNotExist(key TTLStorageImplKey, val string, ttlSeconds int) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.vvmttlstorage.InsertIfNotExists(pKey, cCols, []byte(val), ttlSeconds)
}

func (s *implElectionsITTLStorage) CompareAndSwap(key TTLStorageImplKey, oldVal, newVal string, ttlSeconds int) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.vvmttlstorage.CompareAndSwap(pKey, cCols, []byte(oldVal), []byte(newVal), ttlSeconds)
}

func (s *implElectionsITTLStorage) CompareAndDelete(key TTLStorageImplKey, val string) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.vvmttlstorage.CompareAndDelete(pKey, cCols, []byte(val))
}

func (s *implElectionsITTLStorage) Get(key TTLStorageImplKey) (bool, string, error) {
	pKey, cCols := s.buildKeys(key)
	data := []byte{}
	ok, err := s.vvmttlstorage.Get(pKey, cCols, &data)
	if err != nil {
		return false, "", err
	}
	return ok, string(data), err
}
