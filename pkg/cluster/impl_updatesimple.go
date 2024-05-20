/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/json"
	"fmt"
	"strconv"

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

	fieldsToUpdate := map[string]interface{}{}
	for _, expr := range u.Exprs {
		var val interface{}
		sqlVal := expr.Expr.(*sqlparser.SQLVal)
		switch sqlVal.Type {
		case sqlparser.StrVal:
			val = string(sqlVal.Val)
		case sqlparser.IntVal, sqlparser.FloatVal:
			if val, err = strconv.ParseFloat(string(sqlVal.Val), bitSize64); err != nil {
				// notest
				return err
			}
		case sqlparser.HexNum:
			val = sqlVal.Val
		}
		fieldsToUpdate[expr.Name.Name.String()] = val
	}
	compExpr, ok := u.Where.Expr.(*sqlparser.ComparisonExpr)
	if !ok {
		return errWrongWhere
	}
	if compExpr.Left.(*sqlparser.ColName).Qualifier.Name.String()+appdef.QNameQualifierChar+compExpr.Left.(*sqlparser.ColName).Name.String() != appdef.SystemField_ID {
		return errWrongWhere
	}
	idVal := compExpr.Right.(*sqlparser.SQLVal)
	if idVal.Type != sqlparser.IntVal {
		return errWrongWhere
	}
	id, err := strconv.ParseInt(string(idVal.Val), base10, bitSize64)
	if err != nil {
		// notest: checked already by Type == sqlparserIntVal
		return err
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
