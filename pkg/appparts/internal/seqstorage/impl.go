/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */
package seqstorage

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
)

func (ss *implISeqStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	return ss.events.ReadPLog(ctx, istructs.PartitionID(ss.partitionID), istructs.Offset(offset), istructs.ReadToTheEnd,
		func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			batch := []isequencer.SeqValue{}
			argType := ss.appDef.Type(event.ArgumentObject().QName())

			// odocs
			if argType.Kind() == appdef.TypeKind_ODoc {
				ss.getNumbersFromIObject(event.ArgumentObject(), event.Workspace(), &batch)
			}

			// cuds
			for cud := range event.CUDs {
				if !cud.IsNew() {
					continue
				}
				batch = append(batch, isequencer.SeqValue{
					Key: isequencer.NumberKey{
						WSID:  isequencer.WSID(event.Workspace()),
						SeqID: isequencer.SeqID(ss.seqIDs[cud.QName()]),
					},
					Value: isequencer.Number(cud.ID()),
				})
			}

			return batcher(batch, isequencer.PLogOffset(plogOffset))
		})
}

func (ss *implISeqStorage) WriteValues(batch []isequencer.SeqValue) error {
	const size = 1 + 2 // numbers prefix + size(partitionID)
	cCols := make([]byte, 0, size)
	cCols = append(cCols, cColsNumbers...)
	cCols = binary.BigEndian.AppendUint16(cCols, uint16(ss.partitionID))

	numbersToWriteBytes := []byte{} // FIXME: avoid escape to heap
	ok, err := ss.storage.Get(cCols, &numbersToWriteBytes)
	if err != nil {
		return err
	}
	if !ok {
		numbersToWriteBytes = []byte("{}")
	}

	numbersToWrite := map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{}
	if err := json.Unmarshal(numbersToWriteBytes, &numbersToWrite); err != nil {
		// notest
		return err
	}

	for _, sv := range batch {
		numbersOfWSID, ok := numbersToWrite[sv.Key.WSID]
		if !ok {
			numbersOfWSID = map[isequencer.SeqID]isequencer.Number{}
			numbersToWrite[sv.Key.WSID] = numbersOfWSID
		}
		numbersOfWSID[sv.Key.SeqID] = sv.Value
	}

	if numbersToWriteBytes, err = json.Marshal(&numbersToWrite); err != nil {
		// notest
		return err
	}
	return ss.storage.Put(cCols, numbersToWriteBytes)
}

func (ss *implISeqStorage) ReadNumbers(wsid isequencer.WSID, seqIDs []isequencer.SeqID) ([]isequencer.Number, error) {
	const size = 1 + 3 // numbers prefix + size(partitionID)
	cCols := make([]byte, 0, size)
	cCols = append(cCols, cColsNumbers...)
	cCols = binary.BigEndian.AppendUint16(cCols, uint16(ss.partitionID))
	data := []byte{} // FIXME: avoid escape to heap
	ok, err := ss.storage.Get(cCols, &data)
	if err != nil {
		return nil, err
	}
	if !ok {
		data = []byte("{}")
	}
	storedNumbers := map[isequencer.WSID]map[isequencer.SeqID]isequencer.Number{}
	if err := json.Unmarshal(data, &storedNumbers); err != nil {
		return nil, err
	}

	storedNumbersOfWSID := storedNumbers[wsid]
	if storedNumbersOfWSID == nil {
		storedNumbersOfWSID = map[isequencer.SeqID]isequencer.Number{}
	}
	res := make([]isequencer.Number, len(seqIDs))
	for i, queriedSeqID := range seqIDs {
		res[i] = storedNumbersOfWSID[queriedSeqID]
	}
	return res, nil
}

func (ss *implISeqStorage) WriteNextPLogOffset(offset isequencer.PLogOffset) error {
	const valueSize = 8
	const cColsSize = 1+2 // prefix + partitionID
	cColsBytes := make([]byte, 0, cColsSize)
	cColsBytes = append(cColsBytes, cColsOffset...)
	cColsBytes = binary.BigEndian.AppendUint16(cColsBytes, uint16(ss.partitionID))
	valueBytes := make([]byte, valueSize)
	binary.BigEndian.PutUint64(valueBytes, uint64(offset))
	return ss.storage.Put(cColsBytes, valueBytes)
}

func (ss *implISeqStorage) ReadNextPLogOffset() (isequencer.PLogOffset, error) {
	const valueSize = 8
	const cColsSize = 1+2 // prefix + partitionID
	cColsBytes := make([]byte, 0, cColsSize)
	cColsBytes = append(cColsBytes, cColsOffset...)
	cColsBytes = binary.BigEndian.AppendUint16(cColsBytes, uint16(ss.partitionID))
	valueBytes := make([]byte, valueSize)
	ok, err := ss.storage.Get(cColsBytes, &valueBytes)
	if !ok {
		return 0, err
	}
	offsetUint64 := binary.BigEndian.Uint64(valueBytes)
	return isequencer.PLogOffset(offsetUint64), nil
}

func (ss *implISeqStorage) getNumbersFromIObject(root istructs.IObject, wsid istructs.WSID, batch *[]isequencer.SeqValue) {
	*batch = append(*batch, isequencer.SeqValue{
		Key: isequencer.NumberKey{
			WSID:  isequencer.WSID(wsid),
			SeqID: isequencer.SeqID(ss.seqIDs[root.QName()]),
		},
		Value: isequencer.Number(root.AsRecordID(appdef.SystemField_ID)),
	})
	for container := range root.Containers {
		for c := range root.Children(container) {
			ss.getNumbersFromIObject(c, wsid, batch)
		}
	}
}
