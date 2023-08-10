/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
)

func ProvideBlobberCmds(cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) {
	provideUploadBLOBHelperCmd(cfg)
	provideDownloadBLOBHelperCmd(cfg)
	apps.Parse(schemasFS, appdef.SysPackage, ep)
}

func provideDownloadBLOBHelperCmd(cfg *istructsmem.AppConfigType) {
	dbhQName := appdef.NewQName(appdef.SysPackage, "DownloadBLOBHelper")

	// this command does nothing. It is called to check Authorization token provided in header only
	downloadBLOBHelperCmd := istructsmem.NewCommandFunction(dbhQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec)
	cfg.Resources.Add(downloadBLOBHelperCmd)
}

func provideUploadBLOBHelperCmd(cfg *istructsmem.AppConfigType) {
	uploadBLOBHelperCmd := istructsmem.NewCommandFunction(QNameCommandUploadBLOBHelper, appdef.NullQName, appdef.NullQName, appdef.NullQName, ubhExec)
	cfg.Resources.Add(uploadBLOBHelperCmd)
}

func ubhExec(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	// write a dummy WDoc<BLOB> to book an ID and then use it as a new BLOB ID
	kb, err := args.State.KeyBuilder(state.RecordsStorage, QNameWDocBLOB)
	if err != nil {
		return
	}
	vb, err := args.Intents.NewValue(kb)
	if err != nil {
		return
	}
	vb.PutRecordID(appdef.SystemField_ID, 1)
	vb.PutInt32(fldStatus, int32(iblobstorage.BLOBStatus_Unknown))
	return nil
}
