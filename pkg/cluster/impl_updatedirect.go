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
	if containers, ok := tp.(appdef.IContainers); ok {
		if containers.ContainerCount() > 0 {
			// TODO: no design?
			return errors.New("impossible to update a record that has containers")
		}
	}
	u := stmt.(*sqlparser.Update)
	if tp.Kind() == appdef.TypeKind_ViewRecord {
		if u.Where == nil {
			return errors.New("key condition must be provided for direct update view")
		}
		return updateDirect_View(targetAppStructs, qNameToUpdate, wsid, u, appDef)
	}
	return updateDirect_Record(targetAppStructs, appDef, idToUpdate, wsid, u)
}

func updateDirect_Record(targetAppStructs istructs.IAppStructs, appDef appdef.IAppDef, idToUpdate istructs.RecordID, wsid istructs.WSID, u *sqlparser.Update) error {
	existingRec, err := targetAppStructs.Records().Get(wsid, true, idToUpdate)
	if err != nil {
		return err
	}
	if existingRec.QName() == appdef.NullQName {
		return fmt.Errorf("record ID %d does not exist", idToUpdate)
	}
	newFields, err := getFieldsToUpdate(u.Exprs)
	if err != nil {
		return err
	}
	existingFields := coreutils.FieldsToMap(existingRec, appDef, coreutils.WithNonNilsOnly())
	if err := checkFieldsUpdateAllowed(newFields); err != nil {
		return err
	}
	mergedFields := coreutils.MergeMapsMakeFloats64(existingFields, newFields)
	return targetAppStructs.Records().PutJSON(wsid, mergedFields)
}

func updateDirect_View(targetAppStructs istructs.IAppStructs, qNameToUpdate appdef.QName, wsid istructs.WSID, u *sqlparser.Update, appDef appdef.IAppDef) error {
	keyFields := map[string]interface{}{}
	if err := fillConditionFields(u.Where.Expr, keyFields); err != nil {
		return err
	}

	fieldsToUpdate, err := getFieldsToUpdate(u.Exprs)
	if err != nil {
		// notest
		return err
	}

	if err := checkFieldsUpdateAllowed(fieldsToUpdate); err != nil {
		return err
	}

	kb := targetAppStructs.ViewRecords().KeyBuilder(qNameToUpdate)
	if err := coreutils.MapToObject(keyFields, kb); err != nil {
		return err
	}

	existingViewRec, err := targetAppStructs.ViewRecords().Get(wsid, kb)
	if err != nil {
		// including "not found" error
		return err
	}

	existingFields := coreutils.FieldsToMap(existingViewRec, appDef, coreutils.WithNonNilsOnly())

	mergedFields := coreutils.MergeMapsMakeFloats64(existingFields, fieldsToUpdate, keyFields)
	return targetAppStructs.ViewRecords().PutJSON(wsid, mergedFields)
}

func checkFieldsUpdateAllowed(fieldsToUpdate map[string]interface{}) error {
	for name := range fieldsToUpdate {
		if updateDeniedFields[name] {
			return fmt.Errorf("field %s can not be updated", name)
		}
	}
	return nil
}

func fillConditionFields(expr sqlparser.Expr, fields map[string]interface{}) error {
	switch cond := expr.(type) {
	case *sqlparser.AndExpr:
		if err := fillConditionFields(cond.Left, fields); err != nil {
			return err
		}
		return fillConditionFields(cond.Right, fields)
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
