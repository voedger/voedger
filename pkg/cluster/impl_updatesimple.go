/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func updateSimple(federation federation.IFederation, itokens itokens.ITokens, appQName istructs.AppQName, wsid istructs.WSID, query string, idToUpdate istructs.RecordID) error {
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
	if err := checkFieldsUpdateAllowed(fieldsToUpdate); err != nil {
		return err
	}
	if u.Where != nil {
		return errors.New("where clause is not allowed for update")
	}
	jsonFields, err := json.Marshal(fieldsToUpdate)
	if err != nil {
		// notest
		return err
	}
	cudBody := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":%s}]}`, idToUpdate, jsonFields)
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
		name := expr.Name.Name.String()
		if len(expr.Name.Qualifier.Name.String()) > 0 {
			name = expr.Name.Qualifier.Name.String() + "." + name
		}
		if _, ok := res[name]; ok {
			return nil, fmt.Errorf("field %s specified twice", name)
		}
		res[name] = val
	}
	return res, nil
}

func validateQuery_Simple(sql string) error {
	if len(sql) == 0 {
		return errors.New("empty query")
	}
	return nil
}
