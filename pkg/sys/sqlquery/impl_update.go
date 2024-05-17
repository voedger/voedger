/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sqlquery

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func execCmdVSqlUpdate(args istructs.ExecCommandArgs) (err error) {
	query := args.ArgumentObject.AsString(field_Query)
	appQName, wsid, logViewQName, offset, updateKind, err := parseUpdateQuery(query)
	if err != nil {
		return err
	}
	if appQName == istructs.NullAppQName {
		appQName = args.State.App()
	}
	if wsid == istructs.NullWSID {
		wsid = args.WSID
	}

	switch updateKind {
	case updateKind_Corrupted:
		err = updateCorrupted(appQName, wsid, logViewQName, offset)
	}

	return err
}

func updateCorrupted(appQName istructs.AppQName, wsid istructs.WSID, logViewQName appdef.QName, wlogOffset istructs.Offset, plogOffset istructs.Offset, partitionID istructs.PartitionID,
	currentMillis istructs.UnixMilli) error {
	// read bytes of the existing event
	var as istructs.IAppStructs
	// here we need to read just 1 event - so let's do not consider context of the request
	var currentEventBytes []byte
	as.Events().ReadPLog(context.Background(), partitionID, plogOffset, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		currentEventBytes = event.Bytes()
		return nil
	})
	err := as.Events().ReadWLog(context.Background(), wsid, wlogOffset, 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		currentEventBytes = event.Bytes()
		return nil
	})
	if err != nil {
		return err
	}
	syncRawEventBuilder := as.Events().GetSyncRawEventBuilder(istructs.SyncRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			EventBytes:        currentEventBytes,
			HandlingPartition: partitionID,
			PLogOffset:        plogOffset,
			Workspace:         wsid,
			WLogOffset:        wlogOffset,
			QName:             appdef.NewQName(appdef.SysPackage, "Corrupted"),
			RegisteredAt:      currentMillis,
		},
		SyncedAt: currentMillis,
	})

	syncRawEvent, err := syncRawEventBuilder.BuildRawEvent()
	if err != nil {
		return err
	}
	plogEvent, err := as.Events().PutPlog(syncRawEvent, nil, istructsmem.NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
		panic("must not use ID generator on corrupted event create")
	}))
	if err != nil {
		return err
	}
	return as.Events().PutWlog(plogEvent)
}
