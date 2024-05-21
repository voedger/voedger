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
		`(?P<operation>\s*.+\s+)` + // update, direct update etc
		`(?P<appOwnerAppName>\w+\.\w+\.)` +
		`(?P<wsidOrPartno>\d+\.)` +
		`(?P<qNameToUpdate>\w+\.\w+)` +
		`(?P<idOrOffset>\.\d+)?` +
		`(?P<pars>\s+.*)?` +
		`$`
	bitSize64 = 64
	base10    = 10
)

var (
	QNameViewDeployedApps = appdef.NewQName(ClusterPackage, "DeployedApps")
	qNameWDocApp          = appdef.NewQName(ClusterPackage, "App")
	updateQueryExp        = regexp.MustCompile(updateQueryExpression)
	plog                  = appdef.NewQName(appdef.SysPackage, "PLog")
	wlog                  = appdef.NewQName(appdef.SysPackage, "WLog")
	updateDeniedFields    = map[string]bool{
		appdef.SystemField_ID:    true,
		appdef.SystemField_QName: true,
	}
)

type updateKind int

const (
	updateKind_Null updateKind = iota
	updateKind_Corrupted
	updateKind_Direct
	updateKind_Simple
)
