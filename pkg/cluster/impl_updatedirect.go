/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"fmt"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func updateDirect(asp istructs.IAppStructsProvider, appQName istructs.AppQName, wsid istructs.WSID, qNameToUpdate appdef.QName, appDef appdef.IAppDef,
	query string) error {
	targetAppStructs, err := asp.AppStructs(appQName)
	if err != nil {
		// test here
		return err
	}
	tp := appDef.Type(qNameToUpdate)
	if tp.Kind() == appdef.TypeKind_ViewRecord {
		stmt, err := sqlparser.Parse(query)
		if err != nil {
			return err
		}
		u := stmt.(*sqlparser.Update)

		viewKeyFields := map[string]interface{}{}
		if err := getConditionFields(u.Where.Expr, viewKeyFields); err != nil {
			return err
		}

		kb := targetAppStructs.ViewRecords().KeyBuilder(qNameToUpdate)
		if err := coreutils.MapToObject(viewKeyFields, kb); err != nil {
			return err
		}

		// just to check if all key fields are filled
		if _, err := targetAppStructs.ViewRecords().Get(wsid, kb); err != nil {
			// including "not found" error
			return err
		}

		viewJSON, err := getFieldsToUpdate(u.Exprs)
		if err != nil {
			// notest
			return err
		}
		for k, v := range viewKeyFields {
			viewJSON[k] = v
		}
		viewJSON[appdef.SystemField_QName] = qNameToUpdate.String()
		return targetAppStructs.ViewRecords().PutJSON(wsid, viewJSON)
	} else {
		// WDoc or CDoc only here, parser checked it already
	}
	return nil
}

func getConditionFields(expr sqlparser.Expr, fields map[string]interface{}) error {
	switch cond := expr.(type) {
	case *sqlparser.AndExpr:
		if err := getConditionFields(cond.Left, fields); err != nil {
			return err
		}
		return getConditionFields(cond.Right, fields)
	case *sqlparser.ComparisonExpr:
		if cond.Operator != sqlparser.EqualStr {
			return errWrongWhereForView
		}
		viewKeyColName, ok := cond.Left.(*sqlparser.ColName)
		if !ok {
			return errWrongWhereForView
		}
		fieldName := viewKeyColName.Name.String()
		viewKeySQLVal, ok := cond.Right.(*sqlparser.SQLVal)
		if !ok {
			return errWrongWhereForView
		}
		fieldValue, err := sqlValToInterface(viewKeySQLVal)
		if err != nil {
			// notest
			return err
		}
		if _, ok := fields[fieldName]; ok {
			return fmt.Errorf("key field %s is specified twice", fieldName)
		}
		fields[fieldName] = fieldValue
		return nil
	default:
		return errWrongWhereForView
	}
}

func sqlValToInterface(sqlVal *sqlparser.SQLVal) (val interface{}, err error) {
	switch sqlVal.Type {
	case sqlparser.StrVal:
		return string(sqlVal.Val), nil
	case sqlparser.IntVal, sqlparser.FloatVal:
		if val, err = strconv.ParseFloat(string(sqlVal.Val), bitSize64); err != nil {
			// notest
			return nil, err
		}
		return val, nil
	case sqlparser.HexNum:
		return sqlVal.Val, nil
	default:
		buf := sqlparser.NewTrackedBuffer(nil)
		sqlVal.Format(buf)
		return nil, fmt.Errorf("unsupported sql value: %s, type %d", buf.String(), sqlVal.Type)
	}
}
