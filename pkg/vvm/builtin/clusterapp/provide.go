/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package clusterapp

import (
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
)

func Provide() builtinapps.Builder {
	return func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
		clusterAppPackageFS := parser.PackageFS{
			Path: ClusterAppFQN,
			FS:   schemaFS,
		}
		clusterPackageFS := cluster.Provide(cfg, apis.IAppStructsProvider, apis.ITime, apis.IFederation,
			apis.ITokens, apis.SidecarApps)
		sysPackageFS := sysprovide.Provide(cfg)
		return builtinapps.Def{
			AppQName: istructs.AppQName_sys_cluster,
			Packages: []parser.PackageFS{clusterAppPackageFS, clusterPackageFS, sysPackageFS},
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:         ClusterAppNumPartitions,
				EnginePoolSize:   appparts.PoolSize(uint(ClusterAppNumPartitions), 1, uint(ClusterAppNumPartitions), 1),
				NumAppWorkspaces: ClusterAppNumAppWS,
			},
		}
	}
}
