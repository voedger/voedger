/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

type implIelections struct {
	impl_base
}

func (s *implIelections) buildKeys(key TTLStorageImplKey) (pKey, cCols []byte) {
	pKey = s.getPKey()
	cCols = make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(cCols, key)
	return
}

func (s *implIelections) InsertIfNotExist(key TTLStorageImplKey, val string, ttlSeconds int) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.sysVVMStorage.InsertIfNotExists(pKey, cCols, []byte(val), ttlSeconds)
}

func (s *implIelections) CompareAndSwap(key TTLStorageImplKey, oldVal, newVal string, ttlSeconds int) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.sysVVMStorage.CompareAndSwap(pKey, cCols, []byte(oldVal), []byte(newVal), ttlSeconds)
}

func (s *implIelections) CompareAndDelete(key TTLStorageImplKey, val string) (bool, error) {
	pKey, cCols := s.buildKeys(key)
	return s.sysVVMStorage.CompareAndDelete(pKey, cCols, []byte(val))
}

func (s *implIelections) Get(key TTLStorageImplKey) (bool, string, error) {
	pKey, cCols := s.buildKeys(key)
	data := []byte{}
	ok, err := s.sysVVMStorage.Get(pKey, cCols, &data)
	if err != nil {
		return false, "", err
	}
	return ok, string(data), err
}
