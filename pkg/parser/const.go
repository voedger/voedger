/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package parser

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

const (
	nameCDOC       = "CDoc"
	nameODOC       = "ODoc"
	nameWDOC       = "WDoc"
	nameCSingleton = "CSingleton"
	nameWSingleton = "WSingleton"
	nameCRecord    = "CRecord"
	nameORecord    = "ORecord"
	nameWRecord    = "WRecord"
)

const rootWorkspaceName = "Workspace"

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
	return Ident(fmt.Sprintf("%sDescriptor", wsName))
}
