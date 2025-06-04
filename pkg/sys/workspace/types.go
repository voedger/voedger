/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// template ownerID(raw) -> template ownerRecordField -> uploaded blobID to set to ownerRecordField
type blobsMap map[istructs.RecordID]map[string]istructs.RecordID

type WSPostInitFunc func(targetAppQName appdef.AppQName, wsKind appdef.QName, newWSID istructs.WSID, federation federation.IFederation, authToken string) (err error)

type BLOBWorkspaceTemplateField struct {
	iblobstorage.DescrType
	OwnerRecord      appdef.QName
	OwnerRecordField appdef.FieldName
	OwnerRecordRawID istructs.RecordID
	Content          []byte
}
