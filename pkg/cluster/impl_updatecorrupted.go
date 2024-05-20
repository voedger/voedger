/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func updateCorrupted(asp istructs.IAppStructsProvider, appParts appparts.IAppPartitions, appQName istructs.AppQName, wsidOrPartitionID uint64, logViewQName appdef.QName, offset istructs.Offset, currentMillis istructs.UnixMilli) (err error) {
	targetAppStructs, err := asp.AppStructs(appQName)
	if err != nil {
		// test here
		return err
	}
	var currentEventBytes []byte
	var wlogOffset istructs.Offset
	var plogOffset istructs.Offset
	var partitionID istructs.PartitionID
	var wsid istructs.WSID
	if logViewQName == plog {
		partitionID = istructs.PartitionID(wsidOrPartitionID)
		plogOffset = offset
		err = targetAppStructs.Events().ReadPLog(context.Background(), partitionID, plogOffset, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			currentEventBytes = event.Bytes()
			wlogOffset = event.WLogOffset()
			wsid = event.Workspace()
			return nil
		})
	} else {
		// wlog
		wsid = istructs.WSID(wsidOrPartitionID)
		wlogOffset = offset
		plogOffset = istructs.NullOffset // ok to set NullOffset on update WLog because we do not have way to know how it was stored, no IWLogEvent.PLogOffset() method
		if partitionID, err = appParts.AppWorkspacePartitionID(appQName, wsid); err != nil {
			return err
		}
		err = targetAppStructs.Events().ReadWLog(context.Background(), wsid, wlogOffset, 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
			currentEventBytes = event.Bytes()
			return nil
		})
	}
	if err != nil {
		// notest
		return err
	}
	syncRawEventBuilder := targetAppStructs.Events().GetSyncRawEventBuilder(istructs.SyncRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			EventBytes:        currentEventBytes,
			HandlingPartition: partitionID,
			PLogOffset:        plogOffset,
			Workspace:         wsid,
			WLogOffset:        wlogOffset,
			QName:             istructs.QNameForCorruptedData,
			RegisteredAt:      currentMillis,
		},
		SyncedAt: currentMillis,
	})

	syncRawEvent, err := syncRawEventBuilder.BuildRawEvent()
	if err != nil {
		// notest
		return err
	}
	plogEvent, err := targetAppStructs.Events().PutPlog(syncRawEvent, nil, istructsmem.NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
		// notest
		panic("must not use ID generator on corrupted event create")
	}))
	if err != nil {
		// notest
		return err
	}
	return targetAppStructs.Events().PutWlog(plogEvent)
}
