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
	// appQName, location, qNameToUpdate, offsetOrID, updateKind, cleanSql, err := parseQuery(query)
	if err != nil {
		return update, err
	}

	if !allowedDMLKinds[update.Kind] {
		return update, errors.New("'update' clause expected")
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
		return update, fmt.Errorf("location must be specified")
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

// `123` or `a1` or `login`
// only one of the fields is not zero-value
type location struct {
	number   uint64
	appWSNum uint64
	login    string
}

// // appStructs could be nil
// func parseQuery(query string) (appQName istructs.AppQName, location location, qNameToUpdate appdef.QName, offsetOrID istructs.IDType,
// 	updateKind updateKind, cleanSql string, err error) {
// 	const (
// 		// 0 is original query

// 		operationIdx int = 1 + iota
// 		appIdx
// 		locationIdx
// 		wsidOrPartitionIDIdx
// 		appWSNumIdx
// 		loginIdx
// 		qNameToUpdateIdx
// 		offsetOrIDIdx
// 		parsIdx

// 		groupsCount
// 	)

// 	parts := updateQueryExp.FindStringSubmatch(query)
// 	if len(parts) != groupsCount {
// 		return istructs.NullAppQName, location, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid query format: %s", query)
// 	}

// 	if appName := parts[appIdx]; appName != "" {
// 		appName = appName[:len(parts[appIdx])-1]
// 		owner, app, err := appdef.ParseQualifiedName(appName, `.`)
// 		if err != nil {
// 			// notest: avoided already by regexp
// 			return istructs.NullAppQName, location, appdef.NullQName, 0, updateKind_Null, "", err
// 		}
// 		appQName = istructs.NewAppQName(owner, app)
// 	}

// 	if locationStr := parts[locationIdx]; locationStr != "" {
// 		locationStr = locationStr[:len(parts[locationIdx])-1]
// 		location, err = parseLocation(locationStr)
// 		if err != nil {
// 			// notest: avoided already by regexp
// 			return istructs.NullAppQName, location, appdef.NullQName, 0, updateKind_Null, "", err
// 		}
// 	}

// 	qNameToUpdateStr := parts[qNameToUpdateIdx]
// 	qNameToUpdate, err = appdef.ParseQName(qNameToUpdateStr)
// 	if err != nil {
// 		// notest: avoided already by regexp
// 		return istructs.NullAppQName, location, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid QName %s: %w", qNameToUpdateStr, err)
// 	}

// 	if offsetStr := parts[offsetOrIDIdx]; len(offsetStr) > 0 {
// 		offsetStr = offsetStr[1:]
// 		offsetInt, err := strconv.Atoi(offsetStr)
// 		if err != nil {
// 			// notest: avoided already by regexp
// 			return istructs.NullAppQName, location, appdef.NullQName, 0, updateKind_Null, "", err
// 		}
// 		offsetOrID = istructs.IDType(offsetInt)
// 	}
// 	cleanSql = strings.TrimSpace(parts[parsIdx])
// 	updateKindStr := strings.TrimSpace(parts[operationIdx])
// 	if len(cleanSql) > 0 {
// 		cleanSql = fmt.Sprintf("update %s %s", qNameToUpdate, cleanSql)
// 	}
// 	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
// 	case "update":
// 		updateKind = coreutils.DMLKind_UpdateTable
// 	case "direct update":
// 		updateKind = coreutils.DMLKind_DirectUpdate
// 	case "update corrupted":
// 		updateKind = coreutils.DMLKind_UpdateCorrupted
// 	case "direct insert":
// 		updateKind = coreutils.DMLKind_DirectInsert
// 	default:
// 		return istructs.NullAppQName, location, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("wrong update kind %s", updateKindStr)
// 	}

// 	return appQName, location, qNameToUpdate, offsetOrID, updateKind, cleanSql, nil
// }

func parseLocation(locationStr string) (location location, err error) {
	switch locationStr[:1] {
	case "a":
		appWSNumStr := locationStr[1:]
		location.appWSNum, err = strconv.ParseUint(appWSNumStr, 0, 0)
	case `"`:
		location.login = locationStr[1 : len(locationStr)-1]
	default:
		location.number, err = strconv.ParseUint(locationStr, 0, 0)
	}
	return location, err
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
