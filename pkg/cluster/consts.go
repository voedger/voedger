/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"embed"

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
)

var (
	QNameViewDeployedApps = appdef.NewQName(ClusterPackage, "DeployedApps")
	qNameWDocApp          = appdef.NewQName(ClusterPackage, "App")
)
