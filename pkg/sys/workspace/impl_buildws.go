/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// everything is validated already
func buildWorkspace(templateName string, ep extensionpoints.IExtensionPoint, wsKind appdef.QName, federation federation.IFederation, newWSID istructs.WSID,
	targetAppQName appdef.AppQName, wsName string, systemPrincipalToken string) (err error) {
	wsTemplateBLOBs, wsTemplateData, err := ValidateTemplate(templateName, ep, wsKind)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}
	if len(wsTemplateData) == 0 {
		return nil
	}

	// upload blobs
	blobsMap, err := uploadBLOBs(wsTemplateBLOBs, federation, targetAppQName, newWSID, systemPrincipalToken)
	if err != nil {
		return fmt.Errorf("blobs uploading failed: %w", err)
	}

	// update IDs in workspace template data with new blobs IDs
	updateBLOBsIDsMap(wsTemplateData, blobsMap)

	cudBody := coreutils.JSONMapToCUDBody(wsTemplateData)
	cudURL := fmt.Sprintf("api/%s/%d/c.sys.CUD", targetAppQName.String(), newWSID)
	if _, err := federation.Func(cudURL, cudBody, coreutils.WithAuthorizeBy(systemPrincipalToken), coreutils.WithDiscardResponse()); err != nil {
		return fmt.Errorf("c.sys.CUD failed: %w", err)
	}
	logger.Info(fmt.Sprintf("workspace %s build completed", wsName))
	return nil
}

func updateBLOBsIDsMap(wsData []map[string]interface{}, blobsMap blobsMap) {
	for _, record := range wsData {
		recordIDIntf := record[appdef.SystemField_ID] // record id existence is checked on validation stage
		recordIDClarified, err := coreutils.ClarifyJSONNumber(recordIDIntf.(json.Number), appdef.DataKind_RecordID)
		if err != nil {
			// notest
			panic(err)
		}
		recordID := recordIDClarified.(istructs.RecordID)
		if fieldsBlobIDs, ok := blobsMap[recordID]; ok {
			for fieldName, blobIDToSet := range fieldsBlobIDs {
				// blob fields existence is checked on validation stage
				record[fieldName] = blobIDToSet
			}
		}
	}
}

func uploadBLOBs(blobs []BLOBWorkspaceTemplateField, fed federation.IFederation, appQName appdef.AppQName, wsid istructs.WSID, principalToken string) (blobsMap, error) {
	res := blobsMap{}
	for _, blob := range blobs {
		logger.Info("workspace build: uploading blob", blob.Name)
		blobReader := iblobstorage.BLOBReader{
			DescrType: iblobstorage.DescrType{
				Name:     blob.Name,
				MimeType: blob.MimeType,
			},
			ReadCloser: io.NopCloser(bytes.NewReader(blob.Content)),
		}
		newBLOBID, err := fed.UploadBLOB(appQName, wsid, blobReader, coreutils.WithAuthorizeBy(principalToken))
		if err != nil {
			return nil, fmt.Errorf("blob %s: %w", blob.Name, err)
		}

		fieldBlobID, ok := res[blob.RecordID]
		if !ok {
			fieldBlobID = map[string]istructs.RecordID{}
			res[blob.RecordID] = fieldBlobID
		}
		fieldBlobID[blob.FieldName] = newBLOBID
		logger.Info(fmt.Sprintf("workspace build: blob %s uploaded and set: [%d][%s]=%d", blob.Name, blob.RecordID, blob.FieldName, newBLOBID))
	}
	return res, nil
}
