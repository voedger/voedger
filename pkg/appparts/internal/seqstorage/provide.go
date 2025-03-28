/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

func New(partitionID istructs.PartitionID, events istructs.IEvents) isequencer.ISeqStorage {
	return &implISeqStorage{
		events:      events,
		partitionID: isequencer.PartitionID(partitionID),
	}
}

type implISeqStorage struct {
	partitionID isequencer.PartitionID
	events      istructs.IEvents
	seqIDs      map[appdef.QName]uint16
	storage     storage.ISysVvmStorage
}

func (ss *implISeqStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	return ss.events.ReadPLog(ctx, istructs.PartitionID(ss.partitionID), istructs.Offset(offset), istructs.ReadToTheEnd,
		func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			batch := []isequencer.SeqValue{}
			var appDef appdef.IAppDef
			argType := appDef.Type(event.ArgumentObject().QName())

			// odocs
			if argType.Kind() == appdef.TypeKind_ODoc {
				ss.updateIDGeneratorFromO(event.ArgumentObject(), appDef.Type, event.Workspace(), &batch)
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

	ss.storage.Put()
}

func (ss *implISeqStorage) updateIDGeneratorFromO(root istructs.IObject, findType appdef.FindType, wsid istructs.WSID, batch *[]isequencer.SeqValue) {
	*batch = append(*batch, isequencer.SeqValue{
		Key: isequencer.NumberKey{
			WSID:  isequencer.WSID(wsid),
			SeqID: isequencer.SeqID(ss.seqIDs[root.QName()]),
		},
		Value: isequencer.Number(root.AsRecordID(appdef.SystemField_ID)),
	})
	for container := range root.Containers {
		for c := range root.Children(container) {
			ss.updateIDGeneratorFromO(c, findType, wsid, batch)
		}
	}
}
