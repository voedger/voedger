/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
)

type implISeqSysVVMStorage struct {
	implStorageBase
}

const cColsSize = 8 + 2

func (s *implISeqSysVVMStorage) Get(appID istructs.ClusterAppID, wsid isequencer.WSID, seqID isequencer.SeqID, data *[]byte) (ok bool, err error) {
	pKey := s.getPKey()
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	cCols := make([]byte, cColsSize)
	binary.BigEndian.PutUint64(cCols, uint64(wsid))
	binary.BigEndian.PutUint16(cCols[8:], uint16(seqID))
	return s.sysVVMStorage.Get(pKey, cCols, data)
}

func (s *implISeqSysVVMStorage) Put(appID istructs.ClusterAppID, wsid isequencer.WSID, seqID isequencer.SeqID, value []byte) (err error) {
	pKey := s.getPKey()
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	cCols := make([]byte, cColsSize)
	binary.BigEndian.PutUint64(cCols, uint64(wsid))
	binary.BigEndian.PutUint16(cCols[8:], uint16(seqID))
	return s.sysVVMStorage.Put(pKey, cCols, value)
}
