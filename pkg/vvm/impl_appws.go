/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"strconv"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func BuildAppWorkspaces(vvm *VVM, vvmConfig *VVMConfig) error {
	for _, appQName := range vvm.VVMApps {
		pLogOffsets := map[istructs.PartitionID]istructs.Offset{}
		wLogOffset := istructs.FirstOffset
		nextRecordID := istructs.FirstBaseRecordID + 1
		as, err := vvm.IAppStructsProvider.AppStructs(appQName)
		if err != nil {
			return err
		}
		for wsNum := 0; istructs.AppWSAmount(wsNum) < as.WSAmount(); wsNum++ {
			appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
			existingCDocWSDesc, err := as.Records().GetSingleton(appWSID, commandprocessor.QNameCDocWorkspaceDescriptor)
			if err != nil {
				return err
			}
			if existingCDocWSDesc.QName() != appdef.NullQName {
				logger.Verbose("app workspace", appQName, wsNum, "(", appWSID, ") inited already")
				continue
			}
			partition := istructs.PartitionID(appWSID % istructs.WSID(vvmConfig.PartitionsCount))
			if _, ok := pLogOffsets[partition]; !ok {
				pLogOffsets[partition] = istructs.FirstOffset
			}
			grebp := istructs.GenericRawEventBuilderParams{
				HandlingPartition: partition,
				Workspace:         appWSID,
				QName:             istructs.QNameCommandCUD,
				RegisteredAt:      istructs.UnixMilli(vvmConfig.TimeFunc().UnixMilli()),
				PLogOffset:        pLogOffsets[partition],
				WLogOffset:        wLogOffset,
			}
			reb := as.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: grebp,
					SyncedAt:                     istructs.UnixMilli(vvmConfig.TimeFunc().UnixMilli()),
				},
			)
			cdocWSDesc := reb.CUDBuilder().Create(commandprocessor.QNameCDocWorkspaceDescriptor)
			cdocWSDesc.PutRecordID(appdef.SystemField_ID, 1)
			cdocWSDesc.PutString(authnz.Field_WSName, "appWS"+strconv.Itoa(wsNum))
			cdocWSDesc.PutQName(authnz.Field_WSKind, authnz.QNameCDoc_WorkspaceKind_AppWorkspace)
			cdocWSDesc.PutInt64(authnz.Field_СreatedAtMs, vvmConfig.TimeFunc().UnixMilli())
			cdocWSDesc.PutInt64(commandprocessor.Field_InitCompletedAtMs, vvmConfig.TimeFunc().UnixMilli())
			rawEvent, err := reb.BuildRawEvent()
			if err != nil {
				return err
			}
			pLogEvent, err := as.Events().PutPlog(rawEvent, nil, func(_ istructs.RecordID, _ appdef.IDef) (storage istructs.RecordID, err error) {
				storage = nextRecordID
				nextRecordID++
				return
			})
			if err != nil {
				return err
			}
			pLogOffsets[partition]++
			if err := as.Records().Apply(pLogEvent); err != nil {
				return err
			}
			if _, err = as.Events().PutWlog(pLogEvent); err != nil {
				return err
			}
			wLogOffset++
			logger.Verbose("app workspace", appQName, wsNum, "(", appWSID, ") initialized")
		}
	}
	return nil
}
