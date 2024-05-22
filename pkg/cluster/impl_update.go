/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

		switch update.kind {
		case updateKind_Simple:
			err = updateSimple(update, federation, itokens)
		case updateKind_Corrupted:
			err = updateCorrupted(update, istructs.UnixMilli(timeFunc().UnixMilli()))
		case updateKind_DirectUpdate, updateKind_DirectInsert:
			err = updateDirect(update)
		}
		return coreutils.WrapSysError(err, http.StatusBadRequest)
	}
}

func parseAndValidateQuery(args istructs.ExecCommandArgs, query string, asp istructs.IAppStructsProvider) (update update, err error) {
	appQName, wsidOrPartitionID, qNameToUpdate, offsetOrID, updateKind, cleanSql, err := parseQuery(query)
	update.kind = updateKind
	update.appQName = appQName
	update.qName = qNameToUpdate
	if err != nil {
		return update, err
	}
	if appQName == istructs.NullAppQName {
		return update, errors.New("appQName must be provided")
	}

	update.appParts = args.Workpiece.(interface {
		AppPartitions() appparts.IAppPartitions
	}).AppPartitions()

	update.appStructs, err = asp.AppStructs(appQName)
	if err != nil {
		// test here
		return update, err
	}

	if updateKind != updateKind_Corrupted {
		tp := update.appStructs.AppDef().Type(update.qName)
		if tp.Kind() == appdef.TypeKind_null {
			return update, fmt.Errorf("qname %s is not found", update.qName)
		}
		update.qNameTypeKind = tp.Kind()
	}

	if len(cleanSql) > 0 {
		stmt, err := sqlparser.Parse(cleanSql)
		if err != nil {
			return update, err
		}
		u := stmt.(*sqlparser.Update)

		if u.Exprs != nil {
			if update.setFields, err = getSets(u.Exprs); err != nil {
				return update, err
			}
		} else {
			return update, errors.New("no fields to update")
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

	switch update.kind {
	case updateKind_Simple, updateKind_DirectUpdate, updateKind_DirectInsert:
		update.wsid = istructs.WSID(wsidOrPartitionID)
		update.id = istructs.RecordID(offsetOrID)
	case updateKind_Corrupted:
		update.offset = istructs.Offset(offsetOrID)
		switch update.qName {
		case plog:
			update.partitionID = istructs.PartitionID(wsidOrPartitionID)
		case wlog:
			update.wsid = istructs.WSID(wsidOrPartitionID)
		}
	}

	return update, validateQuery(update)
}

func validateQuery(update update) error {
	switch update.kind {
	case updateKind_Simple:
		return validateQuery_Simple(update)
	case updateKind_Corrupted:
		return validateQuery_Corrupted(update)
	case updateKind_DirectUpdate, updateKind_DirectInsert:
		return validateQuery_Direct(update)
	default:
		// notest: checked already on sql parse
		panic("unknown operation kind" + fmt.Sprint(update.kind))
	}
}

func parseQuery(query string) (appQName istructs.AppQName, wsidOrPartitionID istructs.IDType, qNameToUpdate appdef.QName, offsetOrID istructs.IDType,
	updateKind updateKind, cleanSql string, err error) {
	const (
		// 0 is original query

		operationIdx int = 1 + iota
		appIdx
		wsidOrPartnoIdx
		qNameToUpdateIdx
		offsetOrIDIdx
		parsIdx

		groupsCount
	)

	parts := updateQueryExp.FindStringSubmatch(query)
	if len(parts) != groupsCount {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid query format: %s", query)
	}

	if appName := parts[appIdx]; appName != "" {
		appName = appName[:len(parts[appIdx])-1]
		owner, app, err := appdef.ParseQualifiedName(appName, `.`)
		if err != nil {
			// notest: avoided already by regexp
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
		appQName = istructs.NewAppQName(owner, app)
	}

	if wsIDStr := parts[wsidOrPartnoIdx]; wsIDStr != "" {
		wsIDStr = wsIDStr[:len(parts[wsidOrPartnoIdx])-1]
		wsID, err := strconv.ParseUint(wsIDStr, 0, 0)
		if err != nil {
			// notest: avoided already by regexp
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
		wsidOrPartitionID = istructs.IDType(wsID)
	}

	qNameToUpdateStr := parts[qNameToUpdateIdx]
	qNameToUpdate, err = appdef.ParseQName(qNameToUpdateStr)
	if err != nil {
		// notest: avoided already by regexp
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid QName %s: %w", qNameToUpdateStr, err)
	}

	if offsetStr := parts[offsetOrIDIdx]; len(offsetStr) > 0 {
		offsetStr = offsetStr[1:]
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
		offsetOrID = istructs.IDType(offsetInt)
	}
	cleanSql = strings.TrimSpace(parts[parsIdx])
	updateKindStr := strings.TrimSpace(parts[operationIdx])
	if len(cleanSql) > 0 {
		cleanSql = fmt.Sprintf("update %s %s", qNameToUpdate, cleanSql)
	}
	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
	case "update":
		updateKind = updateKind_Simple
	case "direct update":
		updateKind = updateKind_DirectUpdate
	case "update corrupted":
		updateKind = updateKind_Corrupted
	case "direct insert":
		updateKind = updateKind_DirectInsert
	default:
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("wrong update kind %s", updateKindStr)
	}

	return appQName, wsidOrPartitionID, qNameToUpdate, offsetOrID, updateKind, cleanSql, nil
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

func getSets(exprs sqlparser.UpdateExprs) (map[string]interface{}, error) {
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
