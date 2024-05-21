/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func updateDirect(asp istructs.IAppStructsProvider, appQName istructs.AppQName, wsid istructs.WSID, qNameToUpdate appdef.QName, appDef appdef.IAppDef,
	query string, idToUpdate istructs.RecordID) error {
	targetAppStructs, err := asp.AppStructs(appQName)
	if err != nil {
		// test here
		return err
	}
	tp := appDef.Type(qNameToUpdate)
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return err
	}
	u := stmt.(*sqlparser.Update)
	if tp.Kind() == appdef.TypeKind_ViewRecord {
		return updateDirect_View(targetAppStructs, qNameToUpdate, wsid, u)
	}
	return updateDirect_Record(targetAppStructs, appDef, idToUpdate, wsid, u)
}

func updateDirect_Record(targetAppStructs istructs.IAppStructs, appDef appdef.IAppDef, idToUpdate istructs.RecordID, wsid istructs.WSID, u *sqlparser.Update) error {
	existingRec, err := targetAppStructs.Records().Get(wsid, true, idToUpdate)
	if err != nil {
		// including "not found" error
		return err
	}
	existingFields := coreutils.FieldsToMap(existingRec, appDef, coreutils.WithNonNilsOnly())
	newFields, err := getFieldsToUpdate(u.Exprs)
	if err != nil {
		return err
	}
	if err := checkFieldsUpdateAllowed(newFields); err != nil {
		return err
	}
	mergedFields := map[string]interface{}{}
	coreutils.MergeMaps(mergedFields, existingFields, newFields)
	mergedFields[appdef.SystemField_ID] = float64(mergedFields[appdef.SystemField_ID].(istructs.RecordID)) // PutJSON expects sys.ID to be float64
	return targetAppStructs.Records().PutJSON(wsid, mergedFields)
}

func updateDirect_View(targetAppStructs istructs.IAppStructs, qNameToUpdate appdef.QName, wsid istructs.WSID, u *sqlparser.Update) error {
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
	if err := checkFieldsUpdateAllowed(viewJSON); err != nil {
		return err
	}
	for k, v := range viewKeyFields {
		viewJSON[k] = v
	}
	viewJSON[appdef.SystemField_QName] = qNameToUpdate.String()
	return targetAppStructs.ViewRecords().PutJSON(wsid, viewJSON)
}

func checkFieldsUpdateAllowed(fieldsToUpdate map[string]interface{}) error {
	for name := range fieldsToUpdate {
		if updateDeniedFields[name] {
			return fmt.Errorf("field %s can not be updated", name)
		}
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

func validateQuery_Direct(appDef appdef.IAppDef, qNameToUpdate appdef.QName, offsetOrID istructs.IDType) error {
	tp := appDef.Type(qNameToUpdate)
	if tp == appdef.NullType {
		return fmt.Errorf("qname %s is not found", qNameToUpdate)
	}
	if tp.Kind() != appdef.TypeKind_ViewRecord && tp.Kind() != appdef.TypeKind_CDoc && tp.Kind() != appdef.TypeKind_WDoc {
		return fmt.Errorf("provided qname %s is %s but must be View, CDoc or WDoc", qNameToUpdate, tp.Kind().String())
	}
	typeKindToUpdate := appDef.Type(qNameToUpdate).Kind()
	if typeKindToUpdate == appdef.TypeKind_ViewRecord {
		if offsetOrID > 0 {
			return errors.New("record ID must not be provided on view direct update")
		}
	} else {
		if offsetOrID == 0 {
			return errors.New("record ID must be provided on record direct update")
		}
	}
	return nil
}
