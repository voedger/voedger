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
	"github.com/voedger/voedger/pkg/dml"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func updateOrInsertTable(update update, federation federation.IFederation, itokens itokens.ITokens) error {
	if update.Kind == dml.OpKind_InsertTable {
		update.setFields[appdef.SystemField_ID] = 1
	}
	jsonFields, err := json.Marshal(update.setFields)
	if err != nil {
		// notest
		return err
	}
	cudBody := ""
	if update.Kind == dml.OpKind_InsertTable {
		cudBody = fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, jsonFields)
	} else {
		cudBody = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":%s}]}`, update.id, jsonFields)
	}
	sysToken, err := payloads.GetSystemPrincipalToken(itokens, update.AppQName)
	if err != nil {
		// notest
		return err
	}
	_, err = federation.Func(fmt.Sprintf("api/%s/%d/c.sys.CUD", update.AppQName, update.wsid), cudBody,
		coreutils.WithAuthorizeBy(sysToken),
		coreutils.WithDiscardResponse(),
	)
	return err
}

func validateQuery_Table(update update) error {
	if update.Op.Kind == dml.OpKind_InsertTable && update.id != 0 {
		return errors.New("record ID must not be provided on insert table")
	}
	if update.Op.Kind == dml.OpKind_UpdateTable && update.id == 0 {
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
