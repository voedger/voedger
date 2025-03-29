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

func New(partitionID istructs.PartitionID, events istructs.IEvents, appDef appdef.IAppDef,
	seqStorage isequencer.ISeqSysVVMStorage) isequencer.ISeqStorage {
	return &implISeqStorage{
		events:      events,
		partitionID: isequencer.PartitionID(partitionID),
		appDef:      appDef,
		storage:     seqStorage,
	}
}

type implISeqStorage struct {
	partitionID isequencer.PartitionID
	events      istructs.IEvents
	seqIDs      map[appdef.QName]uint16
	storage     isequencer.ISeqSysVVMStorage
	appDef      appdef.IAppDef
}

func (ss *implISeqStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	return ss.events.ReadPLog(ctx, istructs.PartitionID(ss.partitionID), istructs.Offset(offset), istructs.ReadToTheEnd,
		func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			batch := []isequencer.SeqValue{}
			argType := ss.appDef.Type(event.ArgumentObject().QName())

			// odocs
			if argType.Kind() == appdef.TypeKind_ODoc {
				ss.updateIDGeneratorFromO(event.ArgumentObject(), event.Workspace(), &batch)
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
	value, err := json.Marshal(&batch)
	if err != nil {
		// notest
		return err
	}
	return ss.storage.Put(cColsNumbers, value)
}

func (ss *implISeqStorage) ReadNumbers(isequencer.WSID, []isequencer.SeqID) ([]isequencer.Number, error) {
	data := []byte{} // FIXME: avoid escape to heap
	ok, err := ss.storage.Get(cColsNumbers, &data)
	if !ok {
		return nil, err
	}
	res := []isequencer.Number{}
	return res, json.Unmarshal(data, &res)
}

func (ss *implISeqStorage) WriteNextPLogOffset(offset isequencer.PLogOffset) error {
	const size = 8
	offsetBytes := make([]byte, size)
	binary.BigEndian.PutUint64(offsetBytes, uint64(offset))
	return ss.storage.Put(cColsOffset, offsetBytes)
}

func (ss *implISeqStorage) ReadNextPLogOffset() (isequencer.PLogOffset, error) {
	const size = 8
	offsetBytes := make([]byte, size)
	ok, err := ss.storage.Get(cColsOffset, &offsetBytes)
	if !ok {
		return 0, err
	}
	offsetUint64 := binary.BigEndian.Uint64(offsetBytes)
	return isequencer.PLogOffset(offsetUint64), nil
}

func (ss *implISeqStorage) updateIDGeneratorFromO(root istructs.IObject, wsid istructs.WSID, batch *[]isequencer.SeqValue) {
	*batch = append(*batch, isequencer.SeqValue{
		Key: isequencer.NumberKey{
			WSID:  isequencer.WSID(wsid),
			SeqID: isequencer.SeqID(ss.seqIDs[root.QName()]),
		},
		Value: isequencer.Number(root.AsRecordID(appdef.SystemField_ID)),
	})
	for container := range root.Containers {
		for c := range root.Children(container) {
			ss.updateIDGeneratorFromO(c, wsid, batch)
		}
	}
}
