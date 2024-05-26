/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func updateCorrupted(update update, currentMillis istructs.UnixMilli) (err error) {
	var currentEventBytes []byte
	var wlogOffset istructs.Offset
	var wsid istructs.WSID
	var plogOffset istructs.Offset
	var partitionID istructs.PartitionID
	if update.qName == plog {
		plogOffset = update.offset
		partitionID = update.partitionID
		err = update.appStructs.Events().ReadPLog(context.Background(), update.partitionID, update.offset, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			currentEventBytes = event.Bytes()
			wlogOffset = event.WLogOffset()
			wsid = event.Workspace()
			return nil
		})
	} else {
		// wlog
		wsid = update.wsid
		wlogOffset = update.offset
		plogOffset = istructs.NullOffset // ok to set NullOffset on update WLog because we do not have way to know how it was stored, no IWLogEvent.PLogOffset() method
		if partitionID, err = update.appParts.AppWorkspacePartitionID(update.appQName, wsid); err != nil {
			// notest
			return err
		}
		err = update.appStructs.Events().ReadWLog(context.Background(), wsid, wlogOffset, 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
			currentEventBytes = event.Bytes()
			return nil
		})
	}
	if err != nil {
		// notest
		return err
	}
	syncRawEventBuilder := update.appStructs.Events().GetSyncRawEventBuilder(istructs.SyncRawEventBuilderParams{
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
	if update.qName == plog {
		_, err = update.appStructs.Events().PutPlog(syncRawEvent, nil, istructsmem.NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
			// notest
			panic("must not use ID generator on corrupted event create")
		}))
		return err
	}
	nonStoredPLogEvent := update.appStructs.Events().BuildPLogEvent(syncRawEvent)
	return update.appStructs.Events().PutWlog(nonStoredPLogEvent)
}

func validateQuery_Corrupted(update update) error {
	if len(update.key) > 0 || len(update.setFields) > 0 {
		return fmt.Errorf("any params of update corrupted are not allowed: %s", update.sql)
	}
	if update.offset == 0 {
		return errors.New("offset must be provided")
	}
	switch update.qName {
	case wlog:
		if update.wsid == 0 {
			return errors.New("wsid must be provided for update corrupted wlog")
		}
	case plog:
		partno := istructs.NumAppPartitions(update.partitionID)
		partsCount, err := update.appParts.AppPartsCount(update.appQName)
		if err != nil {
			// notest
			return err
		}
		if partno >= partsCount {
			return fmt.Errorf("provided partno %d is out of %d declared by app %s", partno, partsCount, update.appQName)
		}
	default:
		return fmt.Errorf("invalid log view %s, sys.plog or sys.wlog are only allowed", update.qName)
	}
	return nil
}
