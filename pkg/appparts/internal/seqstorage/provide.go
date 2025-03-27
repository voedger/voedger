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
}

func (ss *implISeqStorage) ActualizeSequencesFromPLog(ctx context.Context, offset isequencer.PLogOffset, batcher func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error) error {
	batch := []isequencer.SeqValue{}
	return ss.events.ReadPLog(ctx, istructs.PartitionID(ss.partitionID), istructs.Offset(offset), istructs.ReadToTheEnd, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		var appDef appdef.IAppDef
		argType := appDef.Type(event.ArgumentObject().QName())
		if argType.Kind() == appdef.TypeKind_ODoc {
			event.ArgumentObject().IDs
			batch = append(batch, isequencer.SeqValue{
				Key: isequencer.NumberKey{
					WSID: isequencer.WSID(event.Workspace()),
					SeqID: isequencer.SeqID(ss.seqIDs[argType.QName()]),
				},
				Value: event.ArgumentObject(),
			})
		}
	})
}
