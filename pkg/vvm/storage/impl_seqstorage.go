/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// [~server.design.sequences/cmp.VVMStorageAdapter~impl]
type implVVMSeqStorageAdapter struct {
	implStorageBase
}

const cColsSize = 8 + 2

func (s *implVVMSeqStorageAdapter) Get(appID istructs.ClusterAppID, wsid isequencer.WSID, seqID isequencer.SeqID) (ok bool, number isequencer.Number, err error) {
	pKey := s.getPKey()
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	cCols := make([]byte, cColsSize)
	binary.BigEndian.PutUint64(cCols, uint64(wsid))
	binary.BigEndian.PutUint16(cCols[8:], uint16(seqID))
	data := make([]byte, utils.Uint64Size)
	ok, err = s.sysVVMStorage.Get(pKey, cCols, &data)
	return ok, isequencer.Number(binary.BigEndian.Uint64(data)), err
}

func (s *implVVMSeqStorageAdapter) PutPLogOffset(appID istructs.ClusterAppID, pLogOffset isequencer.PLogOffset) (err error) {
	pKey := s.getPKey()
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	cCols := make([]byte, cColsSize) // first 8 bytes are 0 for NullWSID
	binary.BigEndian.PutUint16(cCols[8:], istructs.QNameIDPLogOffsetSequence)
	pLogOffsetBytes := make([]byte, utils.Uint64Size)
	binary.BigEndian.PutUint64(pLogOffsetBytes, uint64(pLogOffset))
	return s.sysVVMStorage.Put(pKey, cCols, pLogOffsetBytes)
}

func (s *implVVMSeqStorageAdapter) PutBatch(appID istructs.ClusterAppID, batch []isequencer.SeqValue) error {
	vvmStrorageBatch := make([]istorage.BatchItem, len(batch))
	for i, b := range batch {
		if b.Key.SeqID == isequencer.SeqID(istructs.QNameIDPLogOffsetSequence) {
			panic("cannot write value of PLogOffsetSequence")
		}
		pKey := s.getPKey()
		pKey = binary.BigEndian.AppendUint32(pKey, appID)
		cCols := make([]byte, cColsSize)
		binary.BigEndian.PutUint64(cCols, uint64(b.Key.WSID))
		binary.BigEndian.PutUint16(cCols[8:], uint16(b.Key.SeqID))
		numberBytes := make([]byte, utils.Uint64Size)
		binary.BigEndian.PutUint64(numberBytes, uint64(b.Value))
		vvmStrorageBatch[i].PKey = pKey
		vvmStrorageBatch[i].CCols = cCols
		vvmStrorageBatch[i].Value = numberBytes
	}
	return s.sysVVMStorage.PutBatch(vvmStrorageBatch)
}
