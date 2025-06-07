/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/dml"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
)

func provideExecCmdVSqlUpdate(federation federation.IFederation, itokens itokens.ITokens, time timeu.ITime,
	asp istructs.IAppStructsProvider) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		query := args.ArgumentObject.AsString(field_Query)
		update, err := parseAndValidateQuery(args, query, asp)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		switch update.Kind {
		case dml.OpKind_UpdateTable:
			err = updateTable(update, federation, itokens)
		case dml.OpKind_InsertTable:
			err = insertTable(update, federation, itokens, args.State, args.Intents)
		case dml.OpKind_UpdateCorrupted:
			err = updateCorrupted(update, istructs.UnixMilli(time.Now().UnixMilli()))
		case dml.OpKind_UnloggedUpdate, dml.OpKind_UnloggedInsert:
			err = updateUnlogged(update)
		}
		return coreutils.WrapSysError(err, http.StatusBadRequest)
	}
}

func parseAndValidateQuery(args istructs.ExecCommandArgs, query string, asp istructs.IAppStructsProvider) (update update, err error) {
	update.Op, err = dml.ParseQuery(query)
	if err != nil {
		return update, err
	}

	if !allowedOpKinds[update.Kind] {
		return update, errors.New("'update' or 'insert' clause expected")
	}

	update.appParts = args.Workpiece.(interface {
		AppPartitions() appparts.IAppPartitions
	}).AppPartitions()

	if update.appStructs, err = asp.BuiltIn(update.AppQName); err != nil {
		// notest
		return update, err
	}

	var wsid istructs.IDType
	switch update.Workspace.Kind {
	case dml.WorkspaceKind_WSID:
		wsid = istructs.IDType(update.Workspace.ID)
	case dml.WorkspaceKind_PseudoWSID:
		wsid = istructs.IDType(coreutils.GetAppWSID(istructs.WSID(update.Workspace.ID), update.appStructs.NumAppWorkspaces()))
	case dml.WorkspaceKind_AppWSNum:
		wsid = istructs.IDType(istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+istructs.WSID(update.Workspace.ID)))
	default:
		return update, errors.New("location must be specified")
	}

	if update.Kind != dml.OpKind_UpdateCorrupted {
		tp := update.appStructs.AppDef().Type(update.QName)
		if tp.Kind() == appdef.TypeKind_null {
			return update, fmt.Errorf("qname %s is not found", update.QName)
		}
		update.qNameTypeKind = tp.Kind()
	}

	if len(update.CleanSQL) > 0 {
		stmt, err := sqlparser.Parse(update.CleanSQL)
		if err != nil {
			return update, err
		}
		u := stmt.(*sqlparser.Update)

		if u.Exprs != nil {
			if update.setFields, err = getSets(u.Exprs); err != nil {
				return update, err
			}
		}

		if u.Where != nil {
			update.key = map[string]interface{}{}
			if err := fillWhere(u.Where.Expr, update.key); err != nil {
				return update, err
			}
		}
	}

	if err := checkFieldsUpdateAllowed(update.setFields); err != nil {
		return update, err
	}

	switch update.Kind {
	case dml.OpKind_UpdateTable, dml.OpKind_UnloggedUpdate, dml.OpKind_UnloggedInsert, dml.OpKind_InsertTable:
		update.wsid = istructs.WSID(wsid)
		update.id = istructs.RecordID(update.EntityID)
	case dml.OpKind_UpdateCorrupted:
		update.offset = istructs.Offset(update.EntityID)
		switch update.QName {
		case plog:
			numPartitionsDeployed, err := update.appParts.AppPartsCount(update.AppQName)
			if err != nil {
				return update, err
			}
			if wsid >= istructs.IDType(numPartitionsDeployed) {
				return update, fmt.Errorf("provided partno %d is out of %d declared by app %s", wsid, numPartitionsDeployed, update.AppQName)
			}
			update.partitionID = istructs.PartitionID(wsid) // nolint G115 checked above
		case wlog:
			update.wsid = istructs.WSID(wsid)
		}
	}

	return update, validateQuery(update)
}

func validateQuery(update update) error {
	switch update.Kind {
	case dml.OpKind_UpdateTable:
		return validateQuery_UpdateTable(update)
	case dml.OpKind_InsertTable:
		return validateQuery_InsertTable(update)
	case dml.OpKind_UpdateCorrupted:
		return validateQuery_Corrupted(update)
	case dml.OpKind_UnloggedUpdate, dml.OpKind_UnloggedInsert:
		return validateQuery_Unlogged(update)
	default:
		// notest: checked already on sql parse
		panic("unknown operation kind" + fmt.Sprint(update.Kind))
	}
}

func exprToInterface(expr sqlparser.Expr) (val interface{}, err error) {
	switch typed := expr.(type) {
	case *sqlparser.SQLVal:
		switch typed.Type {
		case sqlparser.StrVal:
			return string(typed.Val), nil
		case sqlparser.IntVal, sqlparser.FloatVal:
			return json.Number(string(typed.Val)), nil
		case sqlparser.HexNum:
			hexBytes := typed.Val[2:] // cut `0x` prefix
			val := make([]byte, len(hexBytes)/2)
			bytesLen, err := hex.Decode(val, hexBytes)
			if err != nil {
				return nil, err
			}
			return val[:bytesLen], nil
		}
	case sqlparser.BoolVal:
		return typed, nil
	case *sqlparser.NullVal:
		return nil, errNullValueNoSupported
	}
	buf := sqlparser.NewTrackedBuffer(nil)
	expr.Format(buf)
	return nil, fmt.Errorf("unsupported value type: %s, type %T", buf.String(), expr)
}

func checkFieldsUpdateAllowed(fieldsToUpdate map[string]interface{}) error {
	for name := range fieldsToUpdate {
		if updateDeniedFields[name] {
			return fmt.Errorf("field %s can not be updated", name)
		}
	}
	return nil
}

func fillWhere(expr sqlparser.Expr, fields map[string]interface{}) error {
	switch cond := expr.(type) {
	case *sqlparser.AndExpr:
		if err := fillWhere(cond.Left, fields); err != nil {
			return err
		}
		return fillWhere(cond.Right, fields)
	case *sqlparser.ComparisonExpr:
		if cond.Operator != sqlparser.EqualStr {
			return errWrongWhereForView
		}
		viewKeyColName, ok := cond.Left.(*sqlparser.ColName)
		if !ok {
			return errWrongWhereForView
		}
		fieldName := colNameToQualifiedName(viewKeyColName)
		var viewKeySQLVal *sqlparser.SQLVal
		switch typed := cond.Right.(type) {
		case *sqlparser.SQLVal:
			viewKeySQLVal = typed
		case *sqlparser.NullVal:
			return errNullValueNoSupported
		default:
			return errWrongWhereForView
		}
		fieldValue, err := exprToInterface(viewKeySQLVal)
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

func colNameToQualifiedName(colName *sqlparser.ColName) string {
	q := colName.Qualifier.Name.String()
	if unlowered, ok := sqlFieldNamesUnlowered[q]; ok {
		q = unlowered
	}
	n := colName.Name.String()
	if unlowered, ok := sqlFieldNamesUnlowered[n]; ok {
		n = unlowered
	}
	if len(q) > 0 {
		return q + "." + n
	}
	return n
}

func getSets(exprs sqlparser.UpdateExprs) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	for _, expr := range exprs {
		var val interface{}
		val, err := exprToInterface(expr.Expr)
		if err != nil {
			// notest
			return nil, err
		}
		name := colNameToQualifiedName(expr.Name)
		if _, ok := res[name]; ok {
			return nil, fmt.Errorf("field %s specified twice", name)
		}
		res[name] = val
	}
	return res, nil
}
