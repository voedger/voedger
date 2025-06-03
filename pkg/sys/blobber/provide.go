/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobber

import (
	"context"
	"fmt"

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

func ProvideBlobberCUDValidators(cfg *istructsmem.AppConfigType) {
	cfg.AddEventValidators(func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) (validateErr error) {
		// [~server.blobs/tuc.HandleBLOBReferences~impl]
		usedBLOBIDs := map[istructs.RecordID]istructs.ICUDRow{}
		rawEvent.CUDs(func(cudRow istructs.ICUDRow) bool {
			cudRow.SpecifiedValues(func(f appdef.IField, val any) bool {
				refField, ok := f.(appdef.IRefField)
				if !ok || len(refField.Refs()) == 0 || !refField.Ref(QNameWDocBLOB) {
					return true
				}
				cudBLOBID := val.(istructs.RecordID)
				// read wdoc.BLOB.OwnerRecord and OwnerRecordField by cudOwnerID
				blobRecord, err := appStructs.Records().Get(wsid, true, cudBLOBID)
				if err != nil {
					// notest
					validateErr = err
					return false
				}
				if blobRecord.QName() == appdef.NullQName {
					// notest: will be validated already by ref integrity validator
					panic(fmt.Sprintf("wdoc.sys.BLOB is not found by ID from %s.%s", cudRow.QName(), f.Name()))
				}
				ownerRecord := blobRecord.AsQName(Field_OwnerRecord)
				if ownerRecord == appdef.NullQName {
					// blob created via APIv1 -> skip
					return false
				}
				ownerRecordID := blobRecord.AsRecordID(Field_OwnerRecordID)
				ownerRecordField := blobRecord.AsString(Field_OwnerRecordField)
				if ownerRecordID != istructs.NullRecordID {
					// [~server.blobs/err.BLOBOwnerRecordIDMustBeEmpty~impl]
					validateErr = fmt.Errorf("BLOB ID %d mentioned in CUD %s.%s is used already in %s.%d.%s", cudBLOBID,
						cudRow.QName(), f.Name(), ownerRecord, ownerRecordID, ownerRecordField)
					return false
				}
				if usedICUDRow, ok := usedBLOBIDs[cudBLOBID]; ok {
					// [~server.blobs/err.DuplicateBLOBReference~impl]
					validateErr = fmt.Errorf("BLOB ID %d mentioned in CUD %s.%s is used already in CUD %s.%d", cudBLOBID,
						cudRow.QName(), f.Name(), usedICUDRow.QName(), usedICUDRow.ID())
					return false
				}
				usedBLOBIDs[cudBLOBID] = cudRow
				if ownerRecord != cudRow.QName() || ownerRecordField != f.Name() {
					// [~server.blobs/err.BLOBOwnerRecordMismatch~impl]
					// [~server.blobs/err.BLOBOwnerRecordFieldMismatch~impl]
					validateErr = fmt.Errorf("BLOB ID %d is intended for %s.%s whereas it is being used in %s.%s",
						cudBLOBID, ownerRecord, ownerRecordField, cudRow.QName(), f.Name())
					return false
				}
				return validateErr == nil
			})
			return true
		})
		return validateErr
	})
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
	vb.PutInt32(field_status, int32(iblobstorage.BLOBStatus_Unknown))
	vb.PutQName(Field_OwnerRecord, args.ArgumentObject.AsQName(Field_OwnerRecord))
	vb.PutString(Field_OwnerRecordField, args.ArgumentObject.AsString(Field_OwnerRecordField))
	return nil
}

func provideRegisterTempBLOB(sr istructsmem.IStatelessResources) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(appdef.NewQName(appdef.SysPackage, "RegisterTempBLOB1d"), istructsmem.NullCommandExec))
}
