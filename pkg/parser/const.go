/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import "github.com/voedger/voedger/pkg/appdef"

const (
	nameCdoc       = "CDoc"
	nameODoc       = "ODoc"
	nameWDoc       = "WDoc"
	nameCSingleton = "CSingleton"
	nameWSingleton = "WSingleton"
	nameCRecord    = "CRecord"
	nameORecord    = "ORecord"
	nameWRecord    = "WRecord"

	nameAppWorkspaceWS = "AppWorkspaceWS"
)

const rootWorkspaceName = appdef.SysWorkspaceName // "Workspace"

const ExportedAppsFile = "apps.yaml"
const ExportedPkgFolder = "pkg"

const maxNestedTableContainerOccurrences = 100 // FIXME: 100 container occurrences
const parserLookahead = 10
const VSqlExt = ".vsql"
const SqlExt = ".sql"

var canNotReferenceTo = map[appdef.TypeKind][]appdef.TypeKind{
	appdef.TypeKind_ODoc:       {},
	appdef.TypeKind_ORecord:    {},
	appdef.TypeKind_WDoc:       {appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_WRecord:    {appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_CDoc:       {appdef.TypeKind_WDoc, appdef.TypeKind_WRecord, appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_CRecord:    {appdef.TypeKind_WDoc, appdef.TypeKind_WRecord, appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_ViewRecord: {appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
}

func defaultDescriptorName(wsName string) Ident {
	return Ident(wsName + "Descriptor")
}
