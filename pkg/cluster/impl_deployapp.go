/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/uniques"
	"github.com/voedger/voedger/pkg/sys/workspace"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// wrong to use IAppPartitions to get total NumAppPartition because the app the cmd is called for is not deployed yet
func provideExecDeployApp(asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		appQNameStr := args.ArgumentObject.AsString(Field_AppQName)
		appQName, err := istructs.ParseAppQName(appQNameStr)
		if err != nil {
			// notest
			return err
		}

		if appQName == istructs.AppQName_sys_cluster {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("%s app can not be deployed by c.cluster.DeployApp", istructs.AppQName_sys_cluster))
		}

		clusterAppStructs, err := asp.AppStructs(istructs.AppQName_sys_cluster)
		if err != nil {
			// notest
			return err
		}
		wdocAppRecordID, err := uniques.GetRecordIDByUniqueCombination(args.WSID, qNameWDocApp, clusterAppStructs, map[string]interface{}{
			Field_AppQName: appQNameStr,
		})
		if err != nil {
			return err
		}
		if wdocAppRecordID != istructs.NullRecordID {
			// deployed already -> just return 200 ok for idempotency and to avoid app workspaces reinit
			return nil
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

		vb.PutRecordID(appdef.SystemField_ID, 1)
		vb.PutString(Field_AppQName, appQNameStr)
		numAppWorkspaces := istructs.NumAppWorkspaces(args.ArgumentObject.AsInt32(Field_NumAppWorkspaces))
		numAppPartitions := istructs.NumAppPartitions(args.ArgumentObject.AsInt32(Field_NumPartitions))
		vb.PutInt32(Field_NumAppWorkspaces, int32(numAppWorkspaces))
		vb.PutInt32(Field_NumPartitions, int32(numAppPartitions))

		// deploy app workspaces
		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			return err
		}
		_, err = InitAppWSes(as, numAppWorkspaces, numAppPartitions, istructs.UnixMilli(timeFunc().UnixMilli()))
		return err
	}
}

// returns an array of inited AppWSIDs. Inited already -> AppWSID is not in the array. Need for testing only
func InitAppWSes(as istructs.IAppStructs, numAppWorkspaces istructs.NumAppWorkspaces, numAppPartitions istructs.NumAppPartitions, currentMillis istructs.UnixMilli) ([]istructs.WSID, error) {
	pLogOffsets := map[istructs.PartitionID]istructs.Offset{}
	wLogOffset := istructs.FirstOffset
	res := []istructs.WSID{}
	for wsNum := 0; istructs.NumAppWorkspaces(wsNum) < numAppWorkspaces; wsNum++ {
		appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
		partitionID := coreutils.AppPartitionID(appWSID, numAppPartitions)
		if _, ok := pLogOffsets[partitionID]; !ok {
			pLogOffsets[partitionID] = istructs.FirstOffset
		}
		inited, err := InitAppWS(as, partitionID, appWSID, pLogOffsets[partitionID], wLogOffset, currentMillis)
		if err != nil {
			return nil, err
		}
		pLogOffsets[partitionID]++
		wLogOffset++
		if inited {
			res = append(res, appWSID)
		}
	}
	return res, nil
}

func InitAppWS(as istructs.IAppStructs, partitionID istructs.PartitionID, wsid istructs.WSID, plogOffset, wlogOffset istructs.Offset, currentMillis istructs.UnixMilli) (inited bool, err error) {
	existingCDocWSDesc, err := as.Records().GetSingleton(wsid, authnz.QNameCDocWorkspaceDescriptor)
	if err != nil {
		return false, err
	}
	if existingCDocWSDesc.QName() != appdef.NullQName {
		logger.Verbose("app workspace", as.AppQName(), wsid-wsid.BaseWSID(), "(", wsid, ") inited already")
		return false, nil
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
	cdocWSDesc.PutInt64(workspace.Field_InitCompletedAtMs, int64(currentMillis))
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		return false, err
	}
	// ok to local IDGenerator here. Actual next record IDs will be determined on the partition recovery stage
	pLogEvent, err := as.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		return false, err
	}
	defer pLogEvent.Release()
	if err := as.Records().Apply(pLogEvent); err != nil {
		return false, err
	}
	if err = as.Events().PutWlog(pLogEvent); err != nil {
		return false, err
	}
	logger.Verbose("app workspace", as.AppQName(), wsid.BaseWSID()-istructs.FirstBaseAppWSID, "(", wsid, ") initialized")
	return true, nil
}
