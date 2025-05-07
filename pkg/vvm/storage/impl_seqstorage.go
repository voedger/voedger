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
)

// [~server.design.sequences/cmp.VVMSeqStorageAdapter~impl]
type implVVMSeqStorageAdapter struct {
	sysVVMStorage ISysVvmStorage
}

const (
	numberPKeySize      = 4 + 4 + 8
	numberCColsSize     = 2
	pLogOffsetPKeySize  = 4 + 2
	pLogOffsetCColsSize = 4
)

func (s *implVVMSeqStorageAdapter) GetNumber(appID isequencer.ClusterAppID, wsid isequencer.WSID, seqID isequencer.SeqID) (ok bool, number isequencer.Number, err error) {
	pKey := make([]byte, 0, numberPKeySize)
	pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage_WS)
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	pKey = binary.BigEndian.AppendUint64(pKey, uint64(wsid))

	cCols := make([]byte, numberCColsSize)
	binary.BigEndian.PutUint16(cCols, uint16(seqID))
	data := make([]byte, utils.Uint64Size)
	ok, err = s.sysVVMStorage.Get(pKey, cCols, &data)
	return ok, isequencer.Number(binary.BigEndian.Uint64(data)), err
}

func (s *implVVMSeqStorageAdapter) GetPLogOffset(partitionID isequencer.PartitionID) (ok bool, pLogOffset isequencer.PLogOffset, err error) {
	pKey := make([]byte, 0, pLogOffsetPKeySize)
	cCols := make([]byte, pLogOffsetCColsSize)
	pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage_Part)
	pKey = binary.BigEndian.AppendUint16(pKey, uint16(partitionID))

	data := make([]byte, utils.Uint64Size)
	ok, err = s.sysVVMStorage.Get(pKey, cCols, &data)
	return ok, isequencer.PLogOffset(binary.BigEndian.Uint64(data)), err
}

func (s *implVVMSeqStorageAdapter) PutPLogOffset(partitionID isequencer.PartitionID, pLogOffset isequencer.PLogOffset) error {
	pKey := make([]byte, 0, pLogOffsetPKeySize)
	cCols := make([]byte, pLogOffsetCColsSize)
	pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage_Part)
	pKey = binary.BigEndian.AppendUint16(pKey, uint16(partitionID))
	pLogOffsetBytes := make([]byte, utils.Uint64Size)
	binary.BigEndian.PutUint64(pLogOffsetBytes, uint64(pLogOffset))
	return s.sysVVMStorage.Put(pKey, cCols, pLogOffsetBytes)
}

func (s *implVVMSeqStorageAdapter) PutNumbers(appID isequencer.ClusterAppID, batch []isequencer.SeqValue) error {
	vvmStrorageBatch := make([]istorage.BatchItem, len(batch))
	for i, b := range batch {
		pKey := make([]byte, 0, numberPKeySize)
		pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage_WS)
		pKey = binary.BigEndian.AppendUint32(pKey, appID)
		pKey = binary.BigEndian.AppendUint64(pKey, uint64(b.Key.WSID))

		cCols := make([]byte, numberCColsSize)
		binary.BigEndian.PutUint16(cCols, uint16(b.Key.SeqID))
		numberBytes := make([]byte, utils.Uint64Size)
		binary.BigEndian.PutUint64(numberBytes, uint64(b.Value))
		vvmStrorageBatch[i].PKey = pKey
		vvmStrorageBatch[i].CCols = cCols
		vvmStrorageBatch[i].Value = numberBytes
	}
	return s.sysVVMStorage.PutBatch(vvmStrorageBatch)
}
