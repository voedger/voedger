/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vvm"
)

// everything is validated already
func buildWorkspace(templateName string, epWSTemplates vvm.IEPWSTemplates, wsKind appdef.QName, federationURL *url.URL, newWSID int64,
	targetAppQName istructs.AppQName, wsName string, systemPrincipalToken string) (err error) {
	wsTemplateBLOBs, wsTemplateData, err := ValidateTemplate(templateName, epWSTemplates, wsKind)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}
	if len(wsTemplateData) == 0 {
		return nil
	}

	// upload blobs
	blobsMap, err := uploadBLOBs(wsTemplateBLOBs, federationURL, targetAppQName, newWSID, systemPrincipalToken)
	if err != nil {
		return fmt.Errorf("blobs uploading failed: %w", err)
	}

	// update IDs in workspace template data with new blobs IDs
	updateBLOBsIDsMap(wsTemplateData, blobsMap)

	cc := make([]cud, 0, len(wsTemplateData))
	for _, record := range wsTemplateData {
		c := cud{
			Fields: make(map[string]interface{}),
		}
		for field, value := range record {
			c.Fields[field] = value
		}
		cc = append(cc, c)
	}
	const batchSize = 50
	batches := make([][]cud, len(cc)/batchSize+1)
	for i := 0; i < len(batches); i++ {
		toCopy := (i + 1) * batchSize
		if toCopy > len(cc) {
			toCopy = len(cc)
		}
		batches[i] = cc[i*batchSize : toCopy]
	}

	initCmdURL := fmt.Sprintf("api/%s/%d/c.sys.Init", targetAppQName.String(), newWSID)
	logger.Info(fmt.Sprintf("workspace %s build starting. %d batches to send, url: %s", wsName, len(batches), initCmdURL))
	for batchNum, batch := range batches {
		logger.Info(fmt.Sprintf("workspace %s building: sending batch %d/%d", wsName, batchNum+1, len(batches)))

		bb, err := json.Marshal(cuds{Cuds: batch})
		if err != nil {
			// validated already
			// notest
			return err
		}

		if _, err := utils.FederationFunc(federationURL, initCmdURL, string(bb), coreutils.WithAuthorizeBy(systemPrincipalToken), coreutils.WithDiscardResponse()); err != nil {
			return fmt.Errorf("c.sys.Init failed: %w", err)
		}
	}
	logger.Info(fmt.Sprintf("workspace %s build completed", wsName))
	return nil
}

func updateBLOBsIDsMap(wsData []map[string]interface{}, blobsMap map[int64]map[string]int64) {
	for _, record := range wsData {
		recordIDIntf := record[appdef.SystemField_ID] // record id existence is checked on validation stage
		recordID := int64(recordIDIntf.(float64))
		if fieldsBlobIDs, ok := blobsMap[recordID]; ok {
			for fieldName, blobIDToSet := range fieldsBlobIDs {
				// blob fields existence is checked on validation stage
				record[fieldName] = blobIDToSet
			}
		}
	}
}

func uploadBLOBs(blobs []BLOB, federationURL *url.URL, appQName istructs.AppQName, wsid int64, principalToken string) (blobsMap, error) {
	res := blobsMap{}
	for _, blob := range blobs {
		uploadBLOBURL := fmt.Sprintf("blob/%s/%d?name=%s&mimeType=%s", appQName.String(), wsid, blob.Name, blob.MimeType)
		logger.Info("workspace build: uploading blob", blob.Name, "url:", uploadBLOBURL)

		resp, err := utils.FederationPOST(federationURL, uploadBLOBURL, string(blob.Content), coreutils.WithAuthorizeBy(principalToken))
		if err != nil {
			return nil, fmt.Errorf("blob %s: %w", blob.Name, err)
		}
		newBLOBID, err := strconv.Atoi(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("blob %s: failed to parse the received blobID string: %w", blob.Name, err)
		}

		fieldBlobID, ok := res[int64(blob.RecordID)]
		if !ok {
			fieldBlobID = map[string]int64{}
			res[int64(blob.RecordID)] = fieldBlobID
		}
		fieldBlobID[blob.FieldName] = int64(newBLOBID)
		logger.Info(fmt.Sprintf("workspace build: blob %s uploaded and set: [%d][%s]=%d", blob.Name, blob.RecordID, blob.FieldName, newBLOBID))
	}
	return res, nil
}
