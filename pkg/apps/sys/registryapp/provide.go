/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registryapp

import (
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/registry"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
)

// for historical reason num partitions of sys/registry must be equal to numCP
func Provide(smtpCfg smtp.Cfg, numCP istructs.NumCommandProcessors) apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) apps.BuiltInAppDef {

		// sys package
		sysPackageFS := sysprovide.Provide(cfg)

		// sys/registry resources
		registryPackageFS := registry.Provide(cfg, apis.ITokens, apis.IFederation)
		cfg.AddSyncProjectors(registry.ProvideSyncProjectorLoginIdx())
		registryAppPackageFS := parser.PackageFS{
			Path: RegistryAppFQN,
			FS:   registryAppSchemaFS,
		}

		return apps.BuiltInAppDef{
			AppQName: istructs.AppQName_sys_registry,
			Packages: []parser.PackageFS{sysPackageFS, registryPackageFS, registryAppPackageFS},
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:         istructs.NumAppPartitions(numCP),
				EnginePoolSize:   appparts.PoolSize(int(numCP), DefDeploymentQPCount, int(numCP), DefDeploymentSPCount),
				NumAppWorkspaces: istructs.DefaultNumAppWorkspaces,
			},
		}
	}
}
