/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"encoding/json"
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
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func provideExecCmdVSqlUpdate(federation federation.IFederation, itokens itokens.ITokens, timeFunc coreutils.TimeFunc) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		query := args.ArgumentObject.AsString(field_Query)
		appQName, wsid, qNameToUpdate, offset, updateKind, cleanSql, err := parseUpdateQuery(query)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}
		if appQName == istructs.NullAppQName {
			appQName = args.State.App()
		}
		if wsid == istructs.NullWSID {
			wsid = args.WSID
		}

		switch updateKind {
		case updateKind_Corrupted:
			appParts := args.Workpiece.(interface {
				AppPartitions() appparts.IAppPartitions
			}).AppPartitions()
			partitionID, err := appParts.AppWorkspacePartitionID(appQName, wsid)
			if err != nil {
				return err
			}
			return updateCorrupted(appQName, wsid, qNameToUpdate, offset, istructs.NullOffset, partitionID, istructs.UnixMilli(timeFunc().UnixMilli()))
		case updateKind_Simple:
			if err = updateSimple(federation, itokens, appQName, wsid, cleanSql, qNameToUpdate); err != nil {
				return coreutils.NewHTTPError(http.StatusBadRequest, err)
			}
		}

		return nil
	}
}

func updateCorrupted(appQName istructs.AppQName, wsid istructs.WSID, logViewQName appdef.QName, wlogOffset istructs.Offset, plogOffset istructs.Offset, partitionID istructs.PartitionID,
	currentMillis istructs.UnixMilli) error {
	// read bytes of the existing event
	var as istructs.IAppStructs // take from the workpiece
	// here we need to read just 1 event - so let's do not consider context of the request
	var currentEventBytes []byte
	as.Events().ReadPLog(context.Background(), partitionID, plogOffset, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		// currentEventBytes = event.Bytes()
		// тут есть wlogOffset
		return nil
	})
	err := as.Events().ReadWLog(context.Background(), wsid, wlogOffset, 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		// currentEventBytes = event.Bytes()
		// тут нету plogOffset
		return nil
	})
	if err != nil {
		return err
	}
	syncRawEventBuilder := as.Events().GetSyncRawEventBuilder(istructs.SyncRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			EventBytes:        currentEventBytes,
			HandlingPartition: partitionID,
			PLogOffset:        plogOffset, // ok to set NullOffset on update WLog because we do not have way to know how it was stored, no IWLogEvent.PLogOffset() method
			Workspace:         wsid,
			WLogOffset:        wlogOffset,
			QName:             istructs.QNameForCorruptedData,
			RegisteredAt:      currentMillis,
		},
		SyncedAt: currentMillis,
	})

	syncRawEvent, err := syncRawEventBuilder.BuildRawEvent()
	if err != nil {
		return err
	}
	plogEvent, err := as.Events().PutPlog(syncRawEvent, nil, istructsmem.NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
		panic("must not use ID generator on corrupted event create")
	}))
	if err != nil {
		return err
	}
	return as.Events().PutWlog(plogEvent)
}

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
	if len(compExpr.Left.(*sqlparser.ColName).Qualifier.Name.String()) > 0 {
		return errWrongWhere
	}
	if compExpr.Left.(*sqlparser.ColName).Name.String() != appdef.SystemField_ID {
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

func parseUpdateQuery(query string) (appQName istructs.AppQName, wsid istructs.WSID, qNameToUpdate appdef.QName, offset istructs.Offset,
	updateKind updateKind, cleanSql string, err error) {
	const (
		// 0 is original query

		firstWordIds int = 1 + iota
		updateKindIdx
		appIdx
		wsidIdx
		qNameToUpdateIdx
		offsetIdx
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

	if wsID := parts[wsidIdx]; wsID != "" {
		wsID = wsID[:len(parts[wsidIdx])-1]
		id, err := strconv.ParseUint(wsID, 0, 0)
		if err != nil {
			// notest: avoided already by regexp
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
		wsid = istructs.WSID(id)
	}

	logViewQNameStr := parts[qNameToUpdateIdx]
	qNameToUpdate, err = appdef.ParseQName(logViewQNameStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid log view QName %s: %w", logViewQNameStr, err)
	}

	if offsetStr := parts[offsetIdx]; len(offsetStr) > 0 {
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid offset %s: %w", offsetStr, err)
		}
		offset = istructs.Offset(offsetInt)
	}
	cleanSql = strings.TrimSpace(parts[parsIdx])
	updateKindStr := parts[firstWordIds] + parts[updateKindIdx]
	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
	case "update":
		updateKind = updateKind_Simple
		if len(cleanSql) == 0 {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("nothing to update: %s", query)
		}
		cleanSql = fmt.Sprintf("update %s %s", qNameToUpdate, cleanSql)
	case "direct update":
		updateKind = updateKind_Direct
	case "update corrupted":
		updateKind = updateKind_Corrupted
		if len(cleanSql) > 0 {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("any params of update corrupted are not allowed: %s", query)
		}
		if qNameToUpdate != wlog || qNameToUpdate != plog {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid log view %s, sys.plog or sys.wlog are only allowed", qNameToUpdate)
		}
		if offset <= 0 {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid offset %d: must be >0", offset)
		}
	default:
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("wrong update kind %s", updateKindStr)
	}

	return appQName, wsid, qNameToUpdate, offset, updateKind, cleanSql, nil
}
