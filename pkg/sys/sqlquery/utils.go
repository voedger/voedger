/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package sqlquery

import (
	"fmt"
	"strconv"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// ParseQueryAppWs parses the query string and returns the application name and workspace ID if presents in the query.
// Also, it returns the cleaned query string without the application name and workspace ID.
//
// The query string should have the following format:
//
//	select [appOwner.appName.][wsid.]tableName[ params]
func parseQueryAppWs(query string) (app istructs.AppQName, ws istructs.WSID, clean string, err error) {
	const (
		selectIdx int = 1 + iota
		appIdx
		wsIdx
		tableIdx
		paramsIdx

		groupsCount
	)

	parts := selectQueryExp.FindStringSubmatch(query)
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
