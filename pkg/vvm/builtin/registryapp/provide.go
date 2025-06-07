/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registryapp

import (
	"fmt"
	"math"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/registry"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
)

// for historical reason num partitions of sys/registry must be equal to numCP
func Provide(smtpCfg smtp.Cfg, numCP istructs.NumCommandProcessors) builtinapps.Builder {
	return func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {

		// sys package
		sysPackageFS := sysprovide.Provide(cfg)

		// sys/registry resources
		registryPackageFS := registry.Provide(cfg, apis.ITokens, apis.IFederation)
		cfg.AddSyncProjectors(registry.ProvideSyncProjectorLoginIdx())
		registryAppPackageFS := parser.PackageFS{
			Path: RegistryAppFQN,
			FS:   registryAppSchemaFS,
		}

		if numCP > math.MaxUint16 {
			panic(fmt.Sprintf("number of command processors can not be more than %d because this number will go to NumAppPartitions uint16", math.MaxUint16))
		}

		return builtinapps.Def{
			AppQName: istructs.AppQName_sys_registry,
			Packages: []parser.PackageFS{sysPackageFS, registryPackageFS, registryAppPackageFS},
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:         istructs.NumAppPartitions(numCP), // nolint G115
				EnginePoolSize:   appparts.PoolSize(uint(numCP), DefDeploymentQPCount, uint(numCP), DefDeploymentSPCount),
				NumAppWorkspaces: istructs.DefaultNumAppWorkspaces,
			},
			CacheAppSchemASTInTests: true,
		}
	}
}
