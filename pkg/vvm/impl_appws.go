/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"strconv"

	"github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/workspace"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// TODO: currently AppWorkspace are created automatically at MainClusterID only. Any request to any other ClusterID -> `workspace is not initialized` error
func BuildAppWorkspaces(vvm *VVM, vvmConfig *VVMConfig) error {
	for appQName := range vvmConfig.VVMAppsBuilder {
		pLogOffsets := map[istructs.PartitionID]istructs.Offset{}
		wLogOffset := istructs.FirstOffset
		as, err := vvm.IAppStructsProvider.AppStructs(appQName)
		if err != nil {
			return err
		}
		for wsNum := 0; istructs.AppWSAmount(wsNum) < as.WSAmount(); wsNum++ {
			appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
			existingCDocWSDesc, err := as.Records().GetSingleton(appWSID, authnz.QNameCDocWorkspaceDescriptor)
			if err != nil {
				return err
			}
			if existingCDocWSDesc.QName() != appdef.NullQName {
				logger.Verbose("app workspace", appQName, wsNum, "(", appWSID, ") inited already")
				continue
			}

			// TODO: eliminate this after move BuildAppWorkspaces to AppDeploy stage
			totalAppPartsCount := getAppPartsCount(appQName, vvm.BuiltInAppsPackages)
			partitionID := coreutils.AppPartitionID(appWSID, totalAppPartsCount)
			if _, ok := pLogOffsets[partitionID]; !ok {
				pLogOffsets[partitionID] = istructs.FirstOffset
			}
			grebp := istructs.GenericRawEventBuilderParams{
				HandlingPartition: partitionID,
				Workspace:         appWSID,
				QName:             istructs.QNameCommandCUD,
				RegisteredAt:      istructs.UnixMilli(vvmConfig.TimeFunc().UnixMilli()),
				PLogOffset:        pLogOffsets[partitionID],
				WLogOffset:        wLogOffset,
			}
			reb := as.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: grebp,
					SyncedAt:                     istructs.UnixMilli(vvmConfig.TimeFunc().UnixMilli()),
				},
			)
			cdocWSDesc := reb.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
			cdocWSDesc.PutRecordID(appdef.SystemField_ID, 1)
			cdocWSDesc.PutString(authnz.Field_WSName, "appWS"+strconv.Itoa(wsNum))
			cdocWSDesc.PutQName(authnz.Field_WSKind, authnz.QNameCDoc_WorkspaceKind_AppWorkspace)
			cdocWSDesc.PutInt64(authnz.Field_CreatedAtMs, vvmConfig.TimeFunc().UnixMilli())
			cdocWSDesc.PutInt64(workspace.Field_InitCompletedAtMs, vvmConfig.TimeFunc().UnixMilli())
			rawEvent, err := reb.BuildRawEvent()
			if err != nil {
				return err
			}
			// ok to local IDGenerator here. Actual next record IDs will be determined on the partition recovery stage
			pLogEvent, err := as.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
			if err != nil {
				return err
			}
			defer pLogEvent.Release()
			pLogOffsets[partitionID]++
			if err := as.Records().Apply(pLogEvent); err != nil {
				return err
			}
			if err = as.Events().PutWlog(pLogEvent); err != nil {
				return err
			}
			wLogOffset++
			logger.Verbose("app workspace", appQName, wsNum, "(", appWSID, ") initialized")
		}
	}
	return nil
}

// TODO: eliminate this after move BuildAppWorkspaces to AppDeploy stage
func getAppPartsCount(appQName istructs.AppQName, builtins []BuiltInAppPackages) coreutils.NumAppPartitions {
	for _, bi := range builtins {
		if bi.BuiltInApp.Name != appQName {
			continue
		}
		return bi.NumParts
	}
	// notest
	panic(appQName.String() + " not found in BuiltInAppsPackages")
}
