/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
)

func ProvideBlobberCmds(res istructsmem.StatelessResources) {
	provideUploadBLOBHelperCmd(res)
	provideDownloadBLOBHelperCmd(res)
}

func provideDownloadBLOBHelperCmd(res istructsmem.StatelessResources) {
	dbhQName := appdef.NewQName(appdef.SysPackage, "DownloadBLOBHelper")

	// this command does nothing. It is called to check Authorization token provided in header only
	downloadBLOBHelperCmd := istructsmem.NewCommandFunction(dbhQName, istructsmem.NullCommandExec)
	res.Add(downloadBLOBHelperCmd)
}

func provideUploadBLOBHelperCmd(res istructsmem.StatelessResources) {
	uploadBLOBHelperCmd := istructsmem.NewCommandFunction(QNameCommandUploadBLOBHelper, ubhExec)
	res.Add(uploadBLOBHelperCmd)
}

func ubhExec(args istructs.ExecCommandArgs) (err error) {
	// write a dummy WDoc<BLOB> to book an ID and then use it as a new BLOB ID
	kb, err := args.State.KeyBuilder(state.Record, QNameWDocBLOB)
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
