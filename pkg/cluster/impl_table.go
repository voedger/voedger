/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

func updateTable(update update, federation federation.IFederation, itokens itokens.ITokens) (cudWLogOffset istructs.Offset, err error) {
	jsonFields, err := json.Marshal(update.setFields)
	if err != nil {
		// notest
		return 0, err
	}
	cudBody := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":%s}]}`, update.id, jsonFields)
	sysToken, err := payloads.GetSystemPrincipalToken(itokens, update.AppQName)
	if err != nil {
		// notest
		return 0, err
	}
	resp, err := federation.Func(fmt.Sprintf("api/%s/%d/c.sys.CUD", update.AppQName, update.wsid), cudBody,
		httpu.WithAuthorizeBy(sysToken),
	)
	if err != nil {
		return 0, err
	}
	return resp.CurrentWLogOffset, nil
}

func insertTable(update update, federation federation.IFederation, itokens itokens.ITokens) (cudWLogOffset istructs.Offset, newID istructs.RecordID, err error) {
	update.setFields[appdef.SystemField_ID] = 1
	update.setFields[appdef.SystemField_QName] = update.QName
	jsonFields, err := json.Marshal(update.setFields)
	if err != nil {
		// notest
		return 0, istructs.NullRecordID, err
	}
	cudBody := fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, jsonFields)
	sysToken, err := payloads.GetSystemPrincipalToken(itokens, update.AppQName)
	if err != nil {
		// notest
		return 0, istructs.NullRecordID, err
	}
	resp, err := federation.Func(fmt.Sprintf("api/%s/%d/c.sys.CUD", update.AppQName, update.wsid), cudBody, httpu.WithAuthorizeBy(sysToken))
	if err != nil {
		return 0, istructs.NullRecordID, err
	}
	return resp.CurrentWLogOffset, resp.NewID(), nil
}

func validateQuery_UpdateTable(update update) error {
	if !allowedDocsTypeKinds[update.qNameTypeKind] {
		return errors.New("CDoc or WDoc only expected")
	}
	if update.id == 0 {
		return errors.New("record ID is not provided on update table")
	}
	if len(update.key) > 0 {
		return errors.New("conditions are not allowed on update table")
	}
	if len(update.setFields) == 0 {
		return errors.New("no fields to update")
	}
	return nil
}

func validateQuery_InsertTable(update update) error {
	if !allowedDocsTypeKinds[update.qNameTypeKind] {
		return errors.New("CDoc or WDoc only expected")
	}
	if update.id != 0 {
		return errors.New("record ID must not be provided on insert table")
	}
	if len(update.key) > 0 {
		return errors.New("conditions are not allowed on insert table")
	}
	if len(update.setFields) == 0 {
		return errors.New("no fields to set")
	}
	return nil
}
