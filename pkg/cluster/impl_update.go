/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideExecCmdVSqlUpdate(timeFunc coreutils.TimeFunc) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		query := args.ArgumentObject.AsString(field_Query)
		appQName, wsid, logViewQName, offset, updateKind, cleanSql, err := parseUpdateQuery(query)
		if err != nil {
			return err
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
			err = updateCorrupted(appQName, wsid, logViewQName, offset, istructs.NullOffset, partitionID, istructs.UnixMilli(timeFunc().UnixMilli()))
		case updateKind_Simple:
			err = updateSimple(appQName, wsid, cleanSql)
		}

		return err
	}
}

func updateCorrupted(appQName istructs.AppQName, wsid istructs.WSID, logViewQName appdef.QName, wlogOffset istructs.Offset, plogOffset istructs.Offset, partitionID istructs.PartitionID,
	currentMillis istructs.UnixMilli) error {
	// read bytes of the existing event
	var as istructs.IAppStructs
	// here we need to read just 1 event - so let's do not consider context of the request
	var currentEventBytes []byte
	as.Events().ReadPLog(context.Background(), partitionID, plogOffset, 1, func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		currentEventBytes = event.Bytes()
		return nil
	})
	err := as.Events().ReadWLog(context.Background(), wsid, wlogOffset, 1, func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		currentEventBytes = event.Bytes()
		return nil
	})
	if err != nil {
		return err
	}
	syncRawEventBuilder := as.Events().GetSyncRawEventBuilder(istructs.SyncRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			EventBytes:        currentEventBytes,
			HandlingPartition: partitionID,
			PLogOffset:        plogOffset,
			Workspace:         wsid,
			WLogOffset:        wlogOffset,
			QName:             appdef.NewQName(appdef.SysPackage, "Corrupted"),
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

func updateSimple(appQName istructs.AppQName, wsid istructs.WSID, query string) error {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return err
	}
	u := stmt.(*sqlparser.Update)

	tableName := u.TableExprs[0].(*sqlparser.AliasedTableExpr)
	log.Println(tableName)
	log.Println(u.Exprs)
	return nil
}

func parseUpdateQuery(query string) (appQName istructs.AppQName, wsid istructs.WSID, logViewQName appdef.QName, offset istructs.Offset,
	updateKind updateKind, cleanSql string, err error) {
	const (
		// 0 is original query

		updateKindIdx int = 1 + iota
		appIdx
		wsidIdx
		logViewQNameIdx
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
		own, n, err := appdef.ParseQualifiedName(appName, `.`)
		if err != nil {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
		appQName = istructs.NewAppQName(own, n)
	}

	if wsID := parts[wsidIdx]; wsID != "" {
		wsID = wsID[:len(parts[wsidIdx])-1]
		if id, err := strconv.ParseUint(wsID, 0, 0); err == nil {
			wsid = istructs.WSID(id)
		} else {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", err
		}
	}

	logViewQNameStr := parts[logViewQNameIdx]
	logViewQName, err = appdef.ParseQName(logViewQNameStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid log view QName %s: %w", logViewQNameStr, err)
	}
	if logViewQName != wlog || logViewQName != plog {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid log view %s, sys.plog or sys.wlog are only allowed", logViewQName)
	}

	offsetStr := parts[offsetIdx]
	offsetInt, err := strconv.Atoi(offsetStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid offset %s: %w", offsetStr, err)
	}
	offset = istructs.Offset(offsetInt)
	if offset <= 0 {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid offset %d: must be >0", offset)
	}
	cleanSql = strings.TrimSpace(parts[parsIdx])
	updateKindStr := parts[updateKindIdx]
	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
	case "update":
		updateKind = updateKind_Simple
		if len(cleanSql) == 0 {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("nothing to update: %s", query)
		}
		cleanSql = fmt.Sprintf("update %s %s", logViewQName, cleanSql)
	case "direct update":
		updateKind = updateKind_Direct
	case "update corrupted":
		updateKind = updateKind_Corrupted
		if len(cleanSql) > 0 {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("any params of update corrupted are not allowed: %s", query)
		}
	default:
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("wrong update kind %s", updateKindStr)
	}

	return appQName, wsid, logViewQName, offset, updateKind, cleanSql, nil
}
