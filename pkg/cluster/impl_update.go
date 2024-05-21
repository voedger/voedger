/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
		appQName, wsidOrPartitionID, qNameToUpdate, offsetOrID, updateKind, cleanSql, err := parseUpdateQuery(query)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}
		if appQName == istructs.NullAppQName {
			appQName = args.State.App()
		}
		appParts := args.Workpiece.(interface {
			AppPartitions() appparts.IAppPartitions
		}).AppPartitions()
		appDef, err := appParts.AppDef(appQName)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}
		if err = validateQuery(appQName, appParts, updateKind, cleanSql, qNameToUpdate, wsidOrPartitionID, offsetOrID, appDef); err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		switch updateKind {
		case updateKind_Corrupted:
			return updateCorrupted(asp, appParts, appQName, wsidOrPartitionID, qNameToUpdate, istructs.Offset(offsetOrID), istructs.UnixMilli(timeFunc().UnixMilli()))
		case updateKind_Simple:
			if wsidOrPartitionID == 0 {
				wsidOrPartitionID = istructs.IDType(args.WSID)
			}
			if err = updateSimple(federation, itokens, appQName, istructs.WSID(wsidOrPartitionID), cleanSql, istructs.RecordID(offsetOrID)); err != nil {
				return coreutils.NewHTTPError(http.StatusBadRequest, err)
			}
		case updateKind_Direct:
			return updateDirect(asp, appQName, istructs.WSID(wsidOrPartitionID), qNameToUpdate, appDef, cleanSql, istructs.RecordID(offsetOrID))
		}

		return nil
	}
}

func validateQuery(appQName istructs.AppQName, appparts appparts.IAppPartitions, kind updateKind, sql string, qNameToUpdate appdef.QName,
	wsidOrPartitionID istructs.IDType, offsetOrID istructs.IDType, appDef appdef.IAppDef) error {
	switch kind {
	case updateKind_Simple:
		return validateQuery_Simple(sql)
	case updateKind_Corrupted:
		return validateQuery_Corrupted(appQName, sql, qNameToUpdate, wsidOrPartitionID, offsetOrID, appparts)
	case updateKind_Direct:
		return validateQuery_Direct(appDef, qNameToUpdate, offsetOrID)
	default:
		// notest: checked already on sql parse
		panic("unknown operation kind" + fmt.Sprint(kind))
	}
}

func parseUpdateQuery(query string) (appQName istructs.AppQName, wsidOrPartitionID istructs.IDType, qNameToUpdate appdef.QName, offsetOrID istructs.IDType,
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

	logViewQNameStr := parts[qNameToUpdateIdx]
	qNameToUpdate, err = appdef.ParseQName(logViewQNameStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("invalid log view QName %s: %w", logViewQNameStr, err)
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
	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
	case "update":
		updateKind = updateKind_Simple
		cleanSql = fmt.Sprintf("update %s %s", qNameToUpdate, cleanSql)
	case "direct update":
		updateKind = updateKind_Direct
		cleanSql = fmt.Sprintf("update %s %s", qNameToUpdate, cleanSql)
	case "update corrupted":
		updateKind = updateKind_Corrupted
	default:
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, "", fmt.Errorf("wrong update kind %s", updateKindStr)
	}

	return appQName, wsidOrPartitionID, qNameToUpdate, offsetOrID, updateKind, cleanSql, nil
}
