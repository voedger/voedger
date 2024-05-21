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
		appDef, err := appParts.AppDef(appQName)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}
		if err = validateQuery(appQName, appParts, updateKind, cleanSql, qNameToUpdate, int64(wsidOrPartitionID), offset, appDef); err != nil {
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
		case updateKind_Direct:
			return updateDirect(asp, appQName, istructs.WSID(wsidOrPartitionID), qNameToUpdate, appDef, cleanSql)
		}

		return nil
	}
}

func validateQuery(appQName istructs.AppQName, appparts appparts.IAppPartitions, kind updateKind, sql string, qNameToUpdate appdef.QName,
	wsidOrPartitionID int64, offset istructs.Offset, appDef appdef.IAppDef) error {
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
	case updateKind_Direct:
		tp := appDef.Type(qNameToUpdate)
		if tp == appdef.NullType {
			return fmt.Errorf("qname %s is not found", qNameToUpdate)
		}
		if tp.Kind() != appdef.TypeKind_ViewRecord && tp.Kind() != appdef.TypeKind_CDoc && tp.Kind() != appdef.TypeKind_WDoc {
			return fmt.Errorf("provided qname %s is %s but must be View, CDoc or WDoc", qNameToUpdate, tp.Kind().String())
		}
	}

	return nil
}

func parseUpdateQuery(query string) (appQName istructs.AppQName, wsidOrPartitionID uint64, qNameToUpdate appdef.QName, offset istructs.Offset,
	updateKind updateKind, cleanSql string, err error) {
	const (
		// 0 is original query

		operationIdx int = 1 + iota
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

	return appQName, wsidOrPartitionID, qNameToUpdate, offset, updateKind, cleanSql, nil
}
