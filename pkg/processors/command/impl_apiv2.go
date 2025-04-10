/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

func parseCUDs_v2(cmd *cmdWorkpiece) (err error) {
	switch cmd.cmdMes.Method() {
	case http.MethodPost:
		firstRawID := istructs.MinRawRecordID
		cudNumber := 1
		cmd.parsedCUDs, err = apiV2InsertToCUDs(cmd.requestData, istructs.NullRecordID, &firstRawID, &cudNumber, cmd.cmdMes.QName())
	case http.MethodPatch, http.MethodDelete:
		cmd.parsedCUDs, err = apiV2UpdateToCUDs(cmd)
	default:
		// notest
		panic("unexpected APIv2 http method: " + cmd.cmdMes.Method())
	}
	return err
}

func apiV2UpdateToCUDs(cmd *cmdWorkpiece) (res []parsedCUD, err error) {
	if _, ok := cmd.requestData[appdef.SystemField_ID]; ok {
		return nil, errors.New("sys.ID field is not allowed among fields to update")
	}
	cudXPath := xPath("")
	updateCUD := parsedCUD{
		id:     cmd.cmdMes.DocID(),
		opKind: appdef.OperationKind_Update,
		fields: cmd.requestData,
	}
	if cmd.cmdMes.Method() == http.MethodDelete {
		if len(cmd.requestData) > 0 {
			return nil, errors.New("unexpected body is provided on delete")
		}
		updateCUD.opKind = appdef.OperationKind_Deactivate
		updateCUD.fields = coreutils.MapObject{
			appdef.SystemField_IsActive: false,
		}
	}
	if updateCUD.existingRecord, err = cmd.appStructs.Records().Get(cmd.cmdMes.WSID(), true, istructs.RecordID(updateCUD.id)); err != nil { // nolint G115
		// notest
		return
	}
	if updateCUD.qName = updateCUD.existingRecord.QName(); updateCUD.qName == appdef.NullQName {
		return nil, coreutils.NewHTTPError(http.StatusNotFound, cudXPath.Errorf("record with queried id %d does not exist", updateCUD.id))
	}
	updateCUD.xPath = xPath(fmt.Sprintf("%s %s %s", cudXPath, opKindDesc[updateCUD.opKind], updateCUD.qName))
	res = append(res, updateCUD)
	return res, nil
}

func apiV2InsertToCUDs(requestData coreutils.MapObject, parentSysID istructs.RecordID, nextRawID *istructs.RecordID, cudNumber *int, qName appdef.QName) ([]parsedCUD, error) {
	res := []parsedCUD{}
	cudXPath := xPath("cuds[" + strconv.Itoa(*cudNumber) + "]")
	parsedCUD := parsedCUD{
		qName:  qName,
		fields: coreutils.MapObject{},
		opKind: appdef.OperationKind_Insert,
	}
	res = append(res, parsedCUD)
	rootCUDIdx := len(res) - 1
	sysID, hasExplicitRawID, err := requestData.AsInt64(appdef.SystemField_ID)
	if err != nil {
		return nil, cudXPath.Error(err)
	}
	if hasExplicitRawID {
		parsedCUD.id = istructs.RecordID(sysID)
	} else {
		parsedCUD.id = *nextRawID
		*nextRawID++
	}

	if parentSysID > 0 {
		parsedCUD.fields[appdef.SystemField_ParentID] = istructs.RecordID(parentSysID)
		parsedCUD.fields[appdef.SystemField_Container] = qName.Entity()
	}

	// if the root cdoc has no sys.ID field then any child must not have one
	// any next explicit rawID must not be <nextRawID
	for rootFieldName, rootFieldValue := range requestData {
		if requestCUDChildsIntfs, ok := rootFieldValue.([]interface{}); ok {
			for _, requestCUDChildsIntf := range requestCUDChildsIntfs {
				requestCUDChildsMap := requestCUDChildsIntf.(map[string]interface{})
				parsedCUDsChilds, err := apiV2InsertToCUDs(requestCUDChildsMap, parsedCUD.id, nextRawID /*already bumped here*/, cudNumber,
					appdef.NewQName(qName.Pkg(), rootFieldName))
				if err != nil {
					return nil, err
				}
				res = append(res, parsedCUDsChilds...)
			}
		} else {
			parsedCUD.fields[rootFieldName] = rootFieldValue
		}
	}
	parsedCUD.xPath = xPath(fmt.Sprintf("%s %s %s", cudXPath, opKindDesc[parsedCUD.opKind], parsedCUD.qName))
	res[rootCUDIdx] = parsedCUD
	*cudNumber++
	return res, nil
}
