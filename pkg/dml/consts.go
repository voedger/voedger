/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package dml

import "regexp"

const (
	opRegexpStr = `^` +
		`\s*((\w*\s*update\s*\w*)|(\w*\s*insert)|(select\s+.+\s+from))\s+` +
		`(?P<appQName>[^\d][a-zA-Z0-9_-]+\.[^\d][a-zA-Z0-9_-]+\.)?` +
		`((?P<wsidOrPartno>\d+\.)|(?P<appWSNum>a\d+.)|(?P<login>".+"\.))?` +
		`(?P<qName>[^\d][a-zA-Z0-9_-]+\.[^\d][a-zA-Z0-9_-]+)` +
		`(?P<idOrOffset>\.\d+)?` +
		`(?P<pars>\s+.*)?$`
)

var opRegexp = regexp.MustCompile(opRegexpStr)

const (
	OpKind_Null OpKind = iota
	OpKind_Select
	OpKind_UpdateTable
	OpKind_InsertTable
	OpKind_UnloggedUpdate
	OpKind_UnloggedInsert
	OpKind_UpdateCorrupted
)

const (
	WorkspaceKind_Null WorkspaceKind = iota
	WorkspaceKind_WSID
	WorkspaceKind_PseudoWSID
	WorkspaceKind_AppWSNum
)
