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

func (ss *implISeqStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	return ss.events.ReadPLog(ctx, ss.partitionID, istructs.Offset(offset), istructs.ReadToTheEnd,
		func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			batchProbe := []isequencer.SeqValue{}
			argType := ss.appDef.Type(event.ArgumentObject().QName())

			// odocs
			if argType.Kind() == appdef.TypeKind_ODoc {
				ss.getNumbersFromArgument(event.ArgumentObject(), event.Workspace(), &batchProbe)
			}

			// cuds
			for cud := range event.CUDs {
				if !cud.IsNew() {
					continue
				}
				cudType := ss.appDef.Type(cud.QName())
				seqQName := istructs.QNameCRecordIDSequence
				if cudType.Kind() == appdef.TypeKind_WDoc {
					seqQName = istructs.QNameOWRecordIDSequence
				}
				batchProbe = append(batchProbe, isequencer.SeqValue{
					Key: isequencer.NumberKey{
						WSID:  isequencer.WSID(event.Workspace()),
						SeqID: isequencer.SeqID(ss.seqIDs[seqQName]),
					},
					Value: isequencer.Number(cud.ID()),
				})
			}

			batch := make([]isequencer.SeqValue, 0, len(batchProbe))
			for _, b := range batchProbe {
				if b.Value < isequencer.Number(istructs.MinClusterRecordID) {
					// syncID<322680000000000 -> consider the syncID is from an old template.
					// ignore IDs from external registers
					// see https://github.com/voedger/voedger/issues/688
					// [~server.design.sequences/cmp.appparts.internal.seqStorage.i688~impl]
					continue
				}
				batch = append(batch, b)
			}

			return batcher(batch, isequencer.PLogOffset(plogOffset))
		})
}

func (ss *implISeqStorage) WriteValues(batch []isequencer.SeqValue) error {
	for _, b := range batch {
		numberBytes := make([]byte, sizeInt64)
		binary.BigEndian.PutUint64(numberBytes, uint64(b.Value))
		if err := ss.storage.Put(ss.appID, b.Key.WSID, b.Key.SeqID, numberBytes); err != nil {
			return err
		}
	}
	return nil
}

func (ss *implISeqStorage) ReadNumbers(wsid isequencer.WSID, seqIDs []isequencer.SeqID) ([]isequencer.Number, error) {
	res := make([]isequencer.Number, len(seqIDs))
	for i, seqID := range seqIDs {
		data := make([]byte, sizeInt64)
		ok, err := ss.storage.Get(ss.appID, wsid, seqID, &data)
		if err != nil {
			return nil, err
		}
		if ok {
			res[i] = isequencer.Number(binary.BigEndian.Uint64(data))
		}
	}
	return res, nil
}

func (ss *implISeqStorage) WriteNextPLogOffset(offset isequencer.PLogOffset) error {
	return ss.WriteValues([]isequencer.SeqValue{
		{
			Key: isequencer.NumberKey{
				WSID:  isequencer.WSID(istructs.NullWSID), // for offset
				SeqID: isequencer.SeqID(istructs.QNameIDWLogOffsetSequence),
			},
			Value: isequencer.Number(offset),
		},
	})
}

func (ss *implISeqStorage) ReadNextPLogOffset() (isequencer.PLogOffset, error) {
	numbers, err := ss.ReadNumbers(isequencer.WSID(istructs.NullWSID), []isequencer.SeqID{isequencer.SeqID(istructs.QNameIDWLogOffsetSequence)})
	if err != nil {
		return 0, err
	}
	return isequencer.PLogOffset(numbers[0]), nil
}

func (ss *implISeqStorage) getNumbersFromArgument(root istructs.IObject, wsid istructs.WSID, batch *[]isequencer.SeqValue) {
	*batch = append(*batch, isequencer.SeqValue{
		Key: isequencer.NumberKey{
			WSID:  isequencer.WSID(wsid),
			SeqID: isequencer.SeqID(ss.seqIDs[istructs.QNameOWRecordIDSequence]),
		},
		Value: isequencer.Number(root.AsRecordID(appdef.SystemField_ID)),
	})
	for container := range root.Containers {
		for c := range root.Children(container) {
			ss.getNumbersFromArgument(c, wsid, batch)
		}
	}
}
