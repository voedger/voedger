/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import "github.com/voedger/voedger/pkg/appdef"

const (
	nameCSingleton     = "CSingleton"
	nameWSingleton     = "WSingleton"
	nameAppWorkspaceWS = "AppWorkspaceWS"
)

const rootWorkspaceName = appdef.SysWorkspaceName // "Workspace"

const ExportedAppsFile = "apps.yaml"
const ExportedPkgFolder = "pkg"

const maxNestedTableContainerOccurrences = 100 // FIXME: 100 container occurrences
const parserLookahead = 10
const VSQLExt = ".vsql"
const SQLExt = ".sql"

const OP_SELECT = "SELECT"
const OP_INSERT = "INSERT"
const OP_UPDATE = "UPDATE"
const OP_EXECUTE = "EXECUTE"
const OP_ACTIVATE = "ACTIVATE"
const OP_DEACTIVATE = "DEACTIVATE"

const identifierRegexp = `([a-zA-Z]\w{0,254})|("[a-zA-Z]\w{0,254}")`

var canNotReferenceTo = map[appdef.TypeKind][]appdef.TypeKind{
	appdef.TypeKind_ODoc:       {},
	appdef.TypeKind_ORecord:    {},
	appdef.TypeKind_WDoc:       {appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_WRecord:    {appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_CDoc:       {appdef.TypeKind_WDoc, appdef.TypeKind_WRecord, appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_CRecord:    {appdef.TypeKind_WDoc, appdef.TypeKind_WRecord, appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
	appdef.TypeKind_ViewRecord: {appdef.TypeKind_ODoc, appdef.TypeKind_ORecord},
}

var grantAllToTableOps = []appdef.OperationKind{
	appdef.OperationKind_Select,
	appdef.OperationKind_Insert,
	appdef.OperationKind_Update,
}

func defaultDescriptorName(wsName string) Ident {
	return Ident(wsName + "Descriptor")
}
