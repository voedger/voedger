/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/json"
	"fmt"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func updateSimple(federation federation.IFederation, itokens itokens.ITokens, appQName istructs.AppQName, wsid istructs.WSID, query string, qNameToUpdate appdef.QName) error {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return err
	}
	u := stmt.(*sqlparser.Update)

	fieldsToUpdate, err := getFieldsToUpdate(u.Exprs)
	if err != nil {
		// notest
		return err
	}
	conditionFields := map[string]interface{}{}
	if err := getConditionFields(u.Where.Expr, conditionFields); err != nil {
		return err
	}
	if len(conditionFields) != 1 {
		return errWrongWhere
	}
	idIntf, ok := conditionFields[appdef.SystemField_ID]
	if !ok {
		return errWrongWhere
	}
	id, ok := idIntf.(int64)
	if !ok {
		// notest: checked already by Type == sqlparserIntVal
		return errWrongWhere
	}

	jsonFields, err := json.Marshal(fieldsToUpdate)
	if err != nil {
		// notest
		return err
	}
	cudBody := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":%s}]}`, id, jsonFields)
	sysToken, err := payloads.GetSystemPrincipalToken(itokens, appQName)
	if err != nil {
		// notest
		return err
	}
	_, err = federation.Func(fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, wsid), cudBody,
		coreutils.WithAuthorizeBy(sysToken),
		coreutils.WithDiscardResponse(),
	)
	return err
}

func getFieldsToUpdate(exprs sqlparser.UpdateExprs) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	for _, expr := range exprs {
		var val interface{}
		sqlVal := expr.Expr.(*sqlparser.SQLVal)
		val, err := sqlValToInterface(sqlVal)
		if err != nil {
			// notest
			return nil, err
		}
		res[expr.Name.Name.String()] = val
	}
	return res, nil
}
