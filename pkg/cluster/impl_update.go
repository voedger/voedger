/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func provideExecCmdVSqlUpdate(federation federation.IFederation, itokens itokens.ITokens, timeFunc coreutils.TimeFunc,
	asp istructs.IAppStructsProvider) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		query := args.ArgumentObject.AsString(field_Query)
		update, err := parseAndValidateQuery(args, query, asp)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		switch update.Kind {
		case coreutils.DMLKind_UpdateTable:
			err = updateTable(update, federation, itokens)
		case coreutils.DMLKind_UpdateCorrupted:
			err = updateCorrupted(update, istructs.UnixMilli(timeFunc().UnixMilli()))
		case coreutils.DMLKind_DirectUpdate, coreutils.DMLKind_DirectInsert:
			err = updateDirect(update)
		}
		return coreutils.WrapSysError(err, http.StatusBadRequest)
	}
}

func parseAndValidateQuery(args istructs.ExecCommandArgs, query string, asp istructs.IAppStructsProvider) (update update, err error) {
	update.DML, err = coreutils.ParseQuery(query)
	if err != nil {
		return update, err
	}

	if !allowedDMLKinds[update.Kind] {
		return update, errors.New("'update' or 'insert' clause expected")
	}

	update.appParts = args.Workpiece.(interface {
		AppPartitions() appparts.IAppPartitions
	}).AppPartitions()

	if update.appStructs, err = asp.AppStructs(update.AppQName); err != nil {
		// notest
		return update, err
	}

	var locationID istructs.IDType
	switch update.Location.Kind {
	case coreutils.LocationKind_WSID:
		locationID = istructs.IDType(update.Location.ID)
	case coreutils.LocationKind_PseudoWSID:
		locationID = istructs.IDType(coreutils.GetAppWSID(istructs.WSID(update.Location.ID), update.appStructs.NumAppWorkspaces()))
	case coreutils.LocationKind_AppWSNum:
		locationID = istructs.IDType(istructs.NewWSID(istructs.MainClusterID, istructs.FirstBaseAppWSID+istructs.WSID(update.Location.ID)))
	default:
		return update, errors.New("location must be specified")
	}

	if update.Kind != coreutils.DMLKind_UpdateCorrupted {
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
	case coreutils.DMLKind_UpdateTable, coreutils.DMLKind_DirectUpdate, coreutils.DMLKind_DirectInsert:
		update.wsid = istructs.WSID(locationID)
		update.id = istructs.RecordID(update.EntityID)
	case coreutils.DMLKind_UpdateCorrupted:
		update.offset = istructs.Offset(update.EntityID)
		switch update.QName {
		case plog:
			update.partitionID = istructs.PartitionID(locationID)
		case wlog:
			update.wsid = istructs.WSID(locationID)
		}
	}

	return update, validateQuery(update)
}

func validateQuery(update update) error {
	switch update.Kind {
	case coreutils.DMLKind_UpdateTable:
		return validateQuery_Table(update)
	case coreutils.DMLKind_UpdateCorrupted:
		return validateQuery_Corrupted(update)
	case coreutils.DMLKind_DirectUpdate, coreutils.DMLKind_DirectInsert:
		return validateQuery_Direct(update)
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
			if val, err = strconv.ParseFloat(string(typed.Val), bitSize64); err != nil {
				// notest: avoided already by sql parser
				return nil, err
			}
			return val, nil
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
		viewKeySQLVal, ok := cond.Right.(*sqlparser.SQLVal)
		if !ok {
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
