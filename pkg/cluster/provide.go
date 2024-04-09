/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
)

func ProvideClusterApp(cfg *istructsmem.AppConfigType) (clusterAppPackageFS parser.PackageFS, deploymentDescriptor AppDeploymentDescriptor) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(appdef.NewQName(ClusterPackage, "VSqlQuery"), vsqlQueryExec))
	return parser.PackageFS{
			Path: ClusterAppFQN,
			FS:   schemaSQL,
		}, AppDeploymentDescriptor{
			PartsCount:     1,
			EnginePoolSize: PoolSize(1, 1, 1),
		}
}
