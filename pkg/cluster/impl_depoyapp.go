/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideExecDeployApp(asp istructs.IAppStructsProvider, aps appparts.INumAppPartitionsSource, timeFunc coreutils.TimeFunc) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		appQNameStr := args.ArgumentObject.AsString(Field_AppQName)
		appQName, err := istructs.ParseAppQName(appQNameStr)
		if err != nil {
			// notest
			return err
		}
		kb, err := args.State.KeyBuilder(state.Record, qNameWDocApp)
		if err != nil {
			// notest
			return err
		}
		vb, err := args.Intents.NewValue(kb)
		if err != nil {
			// notest
			return err
		}
		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			return err
		}
		numAppPartitions, err := (*aps).AppPartsCount(appQName)
		if err != nil {
			// notest
			return err
		}
		numAppWorkspaces := as.NumAppWorkspaces()
		vb.PutRecordID(state.Field_ID, 1)
		vb.PutString(Field_AppQName, appQNameStr)

		// deploy app workspaces
		pLogOffsets := map[istructs.PartitionID]istructs.Offset{}
		wLogOffset := istructs.FirstOffset
		for wsNum := 0; istructs.NumAppWorkspaces(wsNum) < numAppWorkspaces; wsNum++ {
			appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
			partitionID := coreutils.AppPartitionID(appWSID, numAppPartitions)
			if _, ok := pLogOffsets[partitionID]; !ok {
				pLogOffsets[partitionID] = istructs.FirstOffset
			}
			if err := InitAppWS(as, partitionID, appWSID, pLogOffsets[partitionID], wLogOffset, istructs.UnixMilli(timeFunc().UnixMilli())); err != nil {
				return err
			}
			pLogOffsets[partitionID]++
			wLogOffset++
		}
		return nil
	}
}

func InitAppWS(as istructs.IAppStructs, partitionID istructs.PartitionID, wsid istructs.WSID, plogOffset, wlogOffset istructs.Offset, currentMillis istructs.UnixMilli) error {
	existingCDocWSDesc, err := as.Records().GetSingleton(wsid, authnz.QNameCDocWorkspaceDescriptor)
	if err != nil {
		return err
	}
	if existingCDocWSDesc.QName() != appdef.NullQName {
		logger.Verbose("app workspace", as.AppQName(), wsid-wsid.BaseWSID(), "(", wsid, ") inited already")
		return nil
	}

	grebp := istructs.GenericRawEventBuilderParams{
		HandlingPartition: partitionID,
		Workspace:         wsid,
		QName:             istructs.QNameCommandCUD,
		RegisteredAt:      currentMillis,
		PLogOffset:        plogOffset,
		WLogOffset:        wlogOffset,
	}
	reb := as.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: grebp,
			SyncedAt:                     currentMillis,
		},
	)
	cdocWSDesc := reb.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
	cdocWSDesc.PutRecordID(appdef.SystemField_ID, 1)
	cdocWSDesc.PutString(authnz.Field_WSName, "appWS0")
	cdocWSDesc.PutQName(authnz.Field_WSKind, authnz.QNameCDoc_WorkspaceKind_AppWorkspace)
	cdocWSDesc.PutInt64(authnz.Field_CreatedAtMs, int64(currentMillis))
	cdocWSDesc.PutInt64(field_InitCompletedAtMs, int64(currentMillis))
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		return err
	}
	// ok to local IDGenerator here. Actual next record IDs will be determined on the partition recovery stage
	pLogEvent, err := as.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		return err
	}
	pLogEvent.Release()
	logger.Verbose("app workspace", as.AppQName(), wsid-wsid.BaseWSID(), "(", wsid, ") initialized")
	return nil
}
