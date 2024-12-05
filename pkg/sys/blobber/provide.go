/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func ProvideBlobberCmds(sr istructsmem.IStatelessResources) {
	provideUploadBLOBHelperCmd(sr)
	provideDownloadBLOBHelperCmd(sr)
	provideDownloadBLOBAuthnzQry(sr)
	provideRegisterTempBLOB(sr)
}

func provideDownloadBLOBAuthnzQry(sr istructsmem.IStatelessResources) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DownloadBLOBAuthnz"),
		istructsmem.NullQueryExec,
	))
}

// Deprecated: use q.sys.DownloadBLOBAuthnz
func provideDownloadBLOBHelperCmd(sr istructsmem.IStatelessResources) {
	dbhQName := appdef.NewQName(appdef.SysPackage, "DownloadBLOBHelper")

	// this command does nothing. It is called to check Authorization token provided in header only
	downloadBLOBHelperCmd := istructsmem.NewCommandFunction(dbhQName, istructsmem.NullCommandExec)
	sr.AddCommands(appdef.SysPackagePath, downloadBLOBHelperCmd)
}

func provideUploadBLOBHelperCmd(sr istructsmem.IStatelessResources) {
	uploadBLOBHelperCmd := istructsmem.NewCommandFunction(QNameCommandUploadBLOBHelper, ubhExec)
	sr.AddCommands(appdef.SysPackagePath, uploadBLOBHelperCmd)
}

func ubhExec(args istructs.ExecCommandArgs) (err error) {
	// write a dummy WDoc<BLOB> to book an ID and then use it as a new BLOB ID
	kb, err := args.State.KeyBuilder(sys.Storage_Record, QNameWDocBLOB)
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

func provideRegisterTempBLOB(sr istructsmem.IStatelessResources) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(appdef.NewQName(appdef.SysPackage, "RegisterTempBLOB1d"), istructsmem.NullCommandExec))
}
