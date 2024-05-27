/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	dmlRegexpStr = `^` +
		`\s*(?P<operation>(\w*\s*update\s*\w*)|(\w*\s*insert)|(select\s+.+\s+from))\s+` +
		`(?P<appQName>[^\d][a-zA-Z0-9_-]+\.[^\d][a-zA-Z0-9_-]+\.)?` +
		`((?P<wsidOrPartno>\d+\.)|(?P<appWSNum>a\d+.)|(?P<login>".+"\.))?` +
		`(?P<qName>[^\d][a-zA-Z0-9_-]+\.[^\d][a-zA-Z0-9_-]+)` +
		`(?P<idOrOffset>\.\d+)?` +
		`(?P<pars>\s+.*)?$`
	bitSize64 = 64
	base10    = 10
)

type DMLKind int

const (
	DMLKind_Null DMLKind = iota
	DMLKind_Select
	DMLKind_UpdateTable
	DMLKind_DirectUpdate
	DMLKind_DirectInsert
	DMLKind_UpdateCorrupted
)

type Location struct {
	ID   uint64
	Kind LocationKind
}

type LocationKind int

const (
	LocationKind_Null LocationKind = iota
	LocationKind_WSID
	LocationKind_PseudoWSID
	LocationKind_AppWSNum
)

type DML struct {
	AppQName istructs.AppQName
	QName    appdef.QName
	Kind     DMLKind
	Location Location
	EntityID istructs.IDType // offset or RecordID
	CleanSQL string
}

var dmlRegexp = regexp.MustCompile(dmlRegexpStr)

func ParseQuery(query string) (dml DML, err error) {
	const (
		// 0 is original query

		operationIdx int = 1 + iota
		operationUpdateIdx
		operationInsertIdx
		operationSelectIdx
		appIdx
		locationIdx
		locationwsidOrPartitionIDIdx
		locationAppWSNumIdx
		locationLloginIdx
		qNameIdx
		offsetOrIDIdx
		parsIdx

		groupsCount
	)

	parts := dmlRegexp.FindStringSubmatch(query)
	if len(parts) != groupsCount {
		return dml, fmt.Errorf("invalid query format: %s", query)
	}

	if appName := parts[appIdx]; appName != "" {
		appName = appName[:len(parts[appIdx])-1]
		owner, app, err := appdef.ParseQualifiedName(appName, `.`)
		if err != nil {
			// notest: avoided already by regexp
			return dml, err
		}
		dml.AppQName = istructs.NewAppQName(owner, app)
	}

	if locationStr := parts[locationIdx]; locationStr != "" {
		locationStr = locationStr[:len(parts[locationIdx])-1]
		dml.Location, err = parseLocation(locationStr)
		if err != nil {
			// notest: avoided already by regexp
			return dml, err
		}
	}

	qNameStr := parts[qNameIdx]
	dml.QName, err = appdef.ParseQName(qNameStr)
	if err != nil {
		// notest: avoided already by regexp
		return dml, fmt.Errorf("invalid QName %s: %w", qNameStr, err)
	}

	if offsetStr := parts[offsetOrIDIdx]; len(offsetStr) > 0 {
		offsetStr = offsetStr[1:]
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil {
			// notest ??
			return dml, err
		}
		dml.EntityID = istructs.IDType(offsetInt)
	}
	pars := strings.TrimSpace(parts[parsIdx])

	operationStr := strings.TrimSpace(parts[operationIdx])
	operationStrLowered := strings.ToLower(operationStr)
	opSQL := "update"
	switch operationStrLowered {
	case "update":
		dml.Kind = DMLKind_UpdateTable
	case "direct update":
		dml.Kind = DMLKind_DirectUpdate
	case "update corrupted":
		dml.Kind = DMLKind_UpdateCorrupted
	case "direct insert":
		dml.Kind = DMLKind_DirectInsert
	default:
		if strings.HasPrefix(operationStrLowered, "select") {
			opSQL = operationStr
			dml.Kind = DMLKind_Select
		} else {
			// notest: avoided already by regexp
			return dml, fmt.Errorf(`wrong dml operation kind "%s"`, operationStr)
		}
	}
	if len(pars) > 0 || dml.Kind == DMLKind_Select {
		dml.CleanSQL = strings.TrimSpace(fmt.Sprintf("%s %s %s", opSQL, qNameStr, pars))
	}
	return dml, nil
}

func parseLocation(locationStr string) (location Location, err error) {
	switch locationStr[:1] {
	case "a":
		appWSNumStr := locationStr[1:]
		location.ID, err = strconv.ParseUint(appWSNumStr, 0, 0)
		location.Kind = LocationKind_AppWSNum
	case `"`:
		login := locationStr[1 : len(locationStr)-1]
		location.ID = uint64(GetPseudoWSID(istructs.NullWSID, login, istructs.MainClusterID))
		location.Kind = LocationKind_PseudoWSID
	default:
		location.ID, err = strconv.ParseUint(locationStr, 0, 0)
		location.Kind = LocationKind_WSID
	}
	return location, err
}
