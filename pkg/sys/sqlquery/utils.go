/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package sqlquery

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// ParseQueryAppWs parses the query string and returns the application name and workspace ID if presents in the query.
// Also, it returns the cleaned query string without the application name and workspace ID.
//
// The query string should have the following format:
//
//	select *|exp[, exp] from [appOwner.appName.][wsid.]tableQName[ params]
func parseQueryAppWs(query string) (app istructs.AppQName, ws istructs.WSID, clean string, err error) {
	const (
		// 0 is original query

		selectIdx int = 1 + iota
		appIdx
		wsIdx
		tableIdx
		paramsIdx

		groupsCount
	)

	parts := updateQueryExp.FindStringSubmatch(query)
	if len(parts) != groupsCount {
		return istructs.NullAppQName, 0, "", fmt.Errorf("invalid query format: %s", query)
	}

	if appName := parts[appIdx]; appName != "" {
		appName = appName[:len(parts[appIdx])-1]
		own, n, err := appdef.ParseQualifiedName(appName, `.`)
		if err != nil {
			return istructs.NullAppQName, 0, "", err
		}
		app = istructs.NewAppQName(own, n)
	}

	if wsID := parts[wsIdx]; wsID != "" {
		wsID = wsID[:len(parts[wsIdx])-1]
		if id, err := strconv.ParseUint(wsID, 0, 0); err == nil {
			ws = istructs.WSID(id)
		} else {
			return istructs.NullAppQName, 0, "", err
		}
	}

	clean = parts[selectIdx] + parts[tableIdx] + parts[paramsIdx]

	return app, ws, clean, nil
}

func parseUpdateQuery(query string) (appQName istructs.AppQName, wsid istructs.WSID, logViewQName appdef.QName, offset istructs.Offset, updateKind updateKind, err error) {
	const (
		// 0 is original query

		updateKindIdx int = 1 + iota
		appIdx
		wsidIdx
		logViewQNameIdx
		offsetIdx

		groupsCount
	)

	parts := updateQueryExp.FindStringSubmatch(query)
	if len(parts) != groupsCount {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, fmt.Errorf("invalid query format: %s", query)
	}

	if appName := parts[appIdx]; appName != "" {
		appName = appName[:len(parts[appIdx])-1]
		own, n, err := appdef.ParseQualifiedName(appName, `.`)
		if err != nil {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, err
		}
		appQName = istructs.NewAppQName(own, n)
	}

	if wsID := parts[wsidIdx]; wsID != "" {
		wsID = wsID[:len(parts[wsidIdx])-1]
		if id, err := strconv.ParseUint(wsID, 0, 0); err == nil {
			wsid = istructs.WSID(id)
		} else {
			return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, err
		}
	}

	logViewQNameStr := parts[logViewQNameIdx]
	logViewQName, err = appdef.ParseQName(logViewQNameStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, fmt.Errorf("invalid log view QName %s: %w", logViewQNameStr, err)
	}
	if logViewQName != wlog || logViewQName != plog {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, fmt.Errorf("invalid log view %s, sys.plog or sys.wlog are only allowed", logViewQName)
	}

	offsetStr := parts[offsetIdx]
	offsetInt, err := strconv.Atoi(offsetStr)
	if err != nil {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, fmt.Errorf("invalid offset %s: %w", offsetStr, err)
	}
	offset = istructs.Offset(offsetInt)
	if offset <= 0 {
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, fmt.Errorf("invalid offset %d: must be >0", offset)
	}
	updateKindStr := parts[updateKindIdx]
	switch strings.TrimSpace(strings.ToLower(updateKindStr)) {
	case "update":
		updateKind = updateKind_Simple
	case "direct update":
		updateKind = updateKind_Direct
	case "update corrupted":
		updateKind = updateKind_Corrupted
	default:
		return istructs.NullAppQName, 0, appdef.NullQName, 0, updateKind_Null, fmt.Errorf("wrong update kind %s", updateKindStr)
	}
	return appQName, wsid, logViewQName, offset, updateKind, nil
}
