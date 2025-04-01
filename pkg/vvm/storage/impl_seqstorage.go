/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

type implISeqSysVVMStorage struct {
	implStorageBase
}

// FIXME: 
func (s *implISeqSysVVMStorage) Get(cCols []byte, data *[]byte) (ok bool, err error) {
	pKey := s.getPKey()
	return s.sysVVMStorage.Get(pKey, cCols, data)
}

func (s *implISeqSysVVMStorage) Put(cCols []byte, value []byte) (err error) {
	pKey := s.getPKey()
	return s.sysVVMStorage.Put(pKey, cCols, value)
}
