/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */
package seqstorage

import (
	"context"
	"encoding/binary"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
)

func (ss *implISeqStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(ctx context.Context, batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	return ss.events.ReadPLog(ctx, ss.partitionID, istructs.Offset(offset), istructs.ReadToTheEnd,
		func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			batch := []isequencer.SeqValue{}
			argType := ss.appDef.Type(event.ArgumentObject().QName())

			// odocs
			if argType.Kind() == appdef.TypeKind_ODoc {
				ss.getNumbersFromObject(event.ArgumentObject(), event.Workspace(), &batch)
			}

			// cuds
			for cud := range event.CUDs {
				if !cud.IsNew() {
					continue
				}
				cudType := ss.appDef.Type(cud.QName())
				seqQName := istructs.QNameCRecordIDSequence
				if cudType.Kind() == appdef.TypeKind_WDoc || cudType.Kind() == appdef.TypeKind_WRecord {
					seqQName = istructs.QNameOWRecordIDSequence
				}
				addToBatch(event.Workspace(), ss.seqIDs[seqQName], cud.ID(), &batch)
			}

			return batcher(ctx, batch, isequencer.PLogOffset(plogOffset))
		})
}

func (ss *implISeqStorage) WriteValuesAndNextPLogOffset(batch []isequencer.SeqValue, pLogOffset isequencer.PLogOffset) error {
	for _, b := range batch {
		if b.Key.SeqID == isequencer.SeqID(istructs.QNameIDPLogOffsetSequence) {
			panic("can not write QNameIDPLogOffsetSequence as value")
		}
		numberBytes := make([]byte, sizeInt64)
		binary.BigEndian.PutUint64(numberBytes, uint64(b.Value))
		if err := ss.storage.Put(ss.appID, b.Key.WSID, b.Key.SeqID, numberBytes); err != nil {
			// notest
			return err
		}
	}

	pLogOffsetBytes := make([]byte, sizeInt64)
	binary.BigEndian.PutUint64(pLogOffsetBytes, uint64(pLogOffset))
	return ss.storage.Put(ss.appID, isequencer.WSID(istructs.NullWSID), isequencer.SeqID(istructs.QNameIDPLogOffsetSequence), pLogOffsetBytes)
}

func (ss *implISeqStorage) ReadNumbers(wsid isequencer.WSID, seqIDs []isequencer.SeqID) ([]isequencer.Number, error) {
	res := make([]isequencer.Number, len(seqIDs))
	for i, seqID := range seqIDs {
		data := make([]byte, sizeInt64)
		ok, err := ss.storage.Get(ss.appID, wsid, seqID, &data)
		if err != nil {
			// notest
			return nil, err
		}
		if ok {
			res[i] = isequencer.Number(binary.BigEndian.Uint64(data))
		}
	}
	return res, nil
}

func (ss *implISeqStorage) ReadNextPLogOffset() (isequencer.PLogOffset, error) {
	numbers, err := ss.ReadNumbers(isequencer.WSID(istructs.NullWSID), []isequencer.SeqID{isequencer.SeqID(istructs.QNameIDPLogOffsetSequence)})
	if err != nil {
		// notest
		return 0, err
	}
	return isequencer.PLogOffset(numbers[0]), nil
}

func (ss *implISeqStorage) getNumbersFromObject(root istructs.IObject, wsid istructs.WSID, batch *[]isequencer.SeqValue) {
	addToBatch(wsid, ss.seqIDs[istructs.QNameOWRecordIDSequence], root.AsRecordID(appdef.SystemField_ID), batch)
	for container := range root.Containers {
		for c := range root.Children(container) {
			ss.getNumbersFromObject(c, wsid, batch)
		}
	}
}

func addToBatch(wsid istructs.WSID, seqQNameID istructs.QNameID, recID istructs.RecordID, batch *[]isequencer.SeqValue) {
	if recID < istructs.MinClusterRecordID {
		// syncID<322680000000000 -> consider the syncID is from an old template
		// ignore IDs from external registers
		// see https://github.com/voedger/voedger/issues/688
		// [~server.design.sequences/cmp.ISeqStorageImplementation.i688~impl]
		return
	}
	*batch = append(*batch, isequencer.SeqValue{
		Key: isequencer.NumberKey{
			WSID:  isequencer.WSID(wsid),
			SeqID: isequencer.SeqID(seqQNameID),
		},
		Value: isequencer.Number(recID),
	})
}
