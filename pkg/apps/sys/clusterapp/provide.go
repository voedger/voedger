/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package clusterapp

import (
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide() apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) apps.BuiltInAppDef {
		clusterAppPackageFS := parser.PackageFS{
			Path: ClusterAppFQN,
			FS:   schemaFS,
		}
		clusterPackageFS := cluster.Provide()
		sysPackageFS := sys.Provide(cfg, smtp.Cfg{}, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
			apis.NumCommandProcessors, nil, apis.IAppStorageProvider)
		return apps.BuiltInAppDef{
			AppQName: istructs.AppQName_sys_cluster,
			Packages: []parser.PackageFS{clusterAppPackageFS, clusterPackageFS, sysPackageFS},
			AppDeploymentDescriptor: cluster.AppDeploymentDescriptor{
				PartsCount:     1,
				EnginePoolSize: cluster.PoolSize(1, 1, 1),
			},
		}
	}
}
