/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"embed"
	"regexp"

	"github.com/voedger/voedger/pkg/appdef"
)

//go:embed appws.vsql
var schemaFS embed.FS

const (
	ClusterPackage         = "cluster"
	ClusterPackageFQN      = "github.com/voedger/voedger/pkg/" + ClusterPackage
	Field_ClusterAppID     = "ClusterAppID"
	Field_AppQName         = "AppQName"
	Field_NumPartitions    = "NumPartitions"
	Field_NumAppWorkspaces = "NumAppWorkspaces"
	field_Query            = "Query"
	updateQueryExpression  = `^` +
		`(?P<updateKind>\s*update\s+(\w+\s+)?)` + // update [something] before the view
		`(?P<app>\w+\.\w+\.)?` + // appOwner.appName (+ trailing dot)
		`(?P<ws>\d+\.)?` + // wsid (+ trailing dot)
		`(?P<table>\w+\.\w+)` + // table qualified name (clean)
		`(?P<offset>\.\d+)?` + // offset
		`(?P<pars>\s+.*)?` + // (leading spaces +) params
		`$`
	bitSize64 = 64
)

var (
	QNameViewDeployedApps = appdef.NewQName(ClusterPackage, "DeployedApps")
	qNameWDocApp          = appdef.NewQName(ClusterPackage, "App")
	updateQueryExp        = regexp.MustCompile(updateQueryExpression)
	plog                  = appdef.NewQName(appdef.SysPackage, "plog")
	wlog                  = appdef.NewQName(appdef.SysPackage, "wlog")
)

type updateKind int

const (
	updateKind_Null updateKind = iota
	updateKind_Corrupted
	updateKind_Direct
	updateKind_Simple
)
