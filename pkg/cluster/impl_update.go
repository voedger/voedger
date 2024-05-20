/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"encoding/json"
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
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func provideExecCmdVSqlUpdate(federation federation.IFederation, itokens itokens.ITokens, timeFunc coreutils.TimeFunc, asp istructs.IAppStructsProvider) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		query := args.ArgumentObject.AsString(field_Query)
		appQName, wsidOrPartitionID, qNameToUpdate, offset, updateKind, cleanSql, err := parseUpdateQuery(query)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}
		if appQName == istructs.NullAppQName {
			appQName = args.State.App()
		}
		appParts := args.Workpiece.(interface {
			AppPartitions() appparts.IAppPartitions
		}).AppPartitions()
		if err = validateQuery(appQName, appParts, updateKind, cleanSql, qNameToUpdate, int64(wsidOrPartitionID), offset); err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		switch updateKind {
		case updateKind_Corrupted:
			return updateCorrupted(asp, appParts, appQName, wsidOrPartitionID, qNameToUpdate, offset, istructs.UnixMilli(timeFunc().UnixMilli()))
		case updateKind_Simple:
			if wsidOrPartitionID == 0 {
				wsidOrPartitionID = uint64(args.WSID)
			}
			if err = updateSimple(federation, itokens, appQName, istructs.WSID(wsidOrPartitionID), cleanSql, qNameToUpdate); err != nil {
				return coreutils.NewHTTPError(http.StatusBadRequest, err)
			}
		}

		return nil
	}
}

func updateCorrupted(asp istructs.IAppStructsProvider, appParts appparts.IAppPartitions, appQName istructs.AppQName, wsidOrPartitionID uint64, logViewQName appdef.QName, offset istructs.Offset, currentMillis istructs.UnixMilli) (err error) {
	// read bytes of the existing event
	// here we need to read just 1 event - so let's do not consider context of the request
	targetAppStructs, err := asp.AppStructs(appQName)
	if err != nil {
		// test here
		return err
	}
	var currentEventBytes []byte
	var wlogOffset istructs.Offset
	var plogOffset istructs.Offset
	var partitionID istructs.PartitionID
	var wsid istructs.WSID
	if logViewQName == plog {
		partitionID = istructs.PartitionID(wsidOrPartitionID)
		plogOffset = offset
		err = targetAppStructs.Events().ReadPLog(context.Background(), partitionID, plogOffset, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			currentEventBytes = event.Bytes()
			wlogOffset = event.WLogOffset()
			wsid = event.Workspace()
			return nil
		})
	} else {
		// wlog
		wsid = istructs.WSID(wsidOrPartitionID)
		wlogOffset = offset
		plogOffset = istructs.NullOffset // ok to set NullOffset on update WLog because we do not have way to know how it was stored, no IWLogEvent.PLogOffset() method
		if partitionID, err = appParts.AppWorkspacePartitionID(appQName, wsid); err != nil {
			return err
		}
		err = targetAppStructs.Events().ReadWLog(context.Background(), wsid, wlogOffset, 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
			currentEventBytes = event.Bytes()
			return nil
		})
	}
	if err != nil {
		// notest
		return err
	}
	syncRawEventBuilder := targetAppStructs.Events().GetSyncRawEventBuilder(istructs.SyncRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			EventBytes:        currentEventBytes,
			HandlingPartition: partitionID,
			PLogOffset:        plogOffset,
			Workspace:         wsid,
			WLogOffset:        wlogOffset,
			QName:             istructs.QNameForCorruptedData,
			RegisteredAt:      currentMillis,
		},
		SyncedAt: currentMillis,
	})

	syncRawEvent, err := syncRawEventBuilder.BuildRawEvent()
	if err != nil {
		// notest
		return err
	}
	plogEvent, err := targetAppStructs.Events().PutPlog(syncRawEvent, nil, istructsmem.NewIDGeneratorWithHook(func(rawID, storageID istructs.RecordID, t appdef.IType) error {
		// notest
		panic("must not use ID generator on corrupted event create")
	}))
	if err != nil {
		// notest
		return err
	}
	return targetAppStructs.Events().PutWlog(plogEvent)
}

func validateQuery(appQName istructs.AppQName, appparts appparts.IAppPartitions, kind updateKind, sql string, qNameToUpdate appdef.QName, wsidOrPartitionID int64, offset istructs.Offset) error {
	switch kind {
	case updateKind_Simple:
		if len(sql) == 0 {
			return errors.New("empty query")
		}
	case updateKind_Corrupted:
		if len(sql) > 0 {
			return fmt.Errorf("any params of update corrupted are not allowed: %s", sql)
		}
		if appQName == istructs.NullAppQName {
			return errors.New("appQName must be provided for UPDATE CORRUPTED")
		}
		if offset == istructs.NullOffset {
			return errors.New("offset >0 must be provided for UPDATE CORRUPTED")
		}
		switch qNameToUpdate {
		case wlog:
			if wsidOrPartitionID == 0 {
				return errors.New("wsid must be provided for UPDATE CORRUPTED wlog")
			}
		case plog:
			if wsidOrPartitionID == 0 {
				return errors.New("partno must be provided for UPDATE CORRUPTED plog")
			}
			partno := istructs.NumAppPartitions(wsidOrPartitionID)
			partsCount, err := appparts.AppPartsCount(appQName)
			if err != nil {
				return err
			}
			if partno >= partsCount {
				return fmt.Errorf("provided partno %d is out of %d declared by app %s", partno, partsCount, appQName)
			}
		default:
			return fmt.Errorf("invalid log view %s, sys.plog or sys.wlog are only allowed", qNameToUpdate)
		}
	}
	return nil
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

func parseUpdateQuery(query string) (appQName istructs.AppQName, wsidOrPartitionID uint64, qNameToUpdate appdef.QName, offset istructs.Offset,
	updateKind updateKind, cleanSql string, err error) {
	const (
		// 0 is original query

		updateId int = 1 + iota
		updateAddId
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
		wsidOrPartitionID, err = strconv.ParseUint(wsID, 0, 0)
		if err != nil {
			// notest: avoided already by regexp
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
	}

	logViewQNameStr := parts[qNameToUpdateIdx]
	qNameToUpdate, err = appdef.ParseQName(logViewQNameStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid log view QName %s: %w", logViewQNameStr, err)
	}

	if offsetStr := parts[offsetIdx]; len(offsetStr) > 0 {
		offsetStr = offsetStr[1:]
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid offset %s: %w", offsetStr, err)
		}
		offset = istructs.Offset(offsetInt)
	}
	cleanSql = strings.TrimSpace(parts[parsIdx])
	updateKindStr := strings.TrimSpace(parts[updateId])
	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
	case "update":
		updateKind = updateKind_Simple
		cleanSql = fmt.Sprintf("update %s %s", qNameToUpdate, cleanSql)
	case "direct update":
		updateKind = updateKind_Direct
	case "update corrupted":
		updateKind = updateKind_Corrupted
	default:
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("wrong update kind %s", updateKindStr)
	}

	return appQName, wsidOrPartitionID, qNameToUpdate, offset, updateKind, cleanSql, nil
}
