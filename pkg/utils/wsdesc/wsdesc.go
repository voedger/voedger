/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package wsdescutil

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func AddWorkspaceDescriptorStubDef(adb appdef.IAppDefBuilder) {
	wsDesc := adb.AddCDoc(authnz.QNameCDocWorkspaceDescriptor)
	wsDesc.
		AddField("WSKind", appdef.DataKind_QName, true).
		AddField("Status", appdef.DataKind_int32, true).
		AddField("InitCompletedAtMs", appdef.DataKind_int64, true).
		AddField("InitError", appdef.DataKind_string, false)

	wsDesc.SetSingleton()
}

func CreateCDocWorkspaceDescriptorStub(as istructs.IAppStructs, partNum istructs.PartitionID, wsid istructs.WSID, wsKind appdef.QName) error {
	now := time.Now()
	grebp := istructs.GenericRawEventBuilderParams{
		HandlingPartition: partNum,
		Workspace:         wsid,
		QName:             istructs.QNameCommandCUD,
		RegisteredAt:      istructs.UnixMilli(now.UnixMilli()),
		PLogOffset:        1,
		WLogOffset:        1,
	}
	reb := as.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: grebp,
			SyncedAt:                     istructs.UnixMilli(now.UnixMilli()),
		},
	)
	cdocWSDesc := reb.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
	cdocWSDesc.PutRecordID(appdef.SystemField_ID, 1)
	cdocWSDesc.PutQName("WSKind", wsKind)
	cdocWSDesc.PutInt32("Status", int32(authnz.WorkspaceStatus_Active))
	cdocWSDesc.PutInt64("InitCompletedAtMs", now.UnixMilli())
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		return err
	}
	pLogEvent, err := as.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		return err
	}
	defer pLogEvent.Release()
	if err := as.Records().Apply(pLogEvent); err != nil {
		return err
	}
	return as.Events().PutWlog(pLogEvent)
}
