/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registryapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/registry"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(smtpCfg smtp.Cfg) apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {

		// sys package
		sys.Provide(cfg, appDefBuilder, smtpCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
			apis.NumCommandProcessors, nil, apis.IAppStorageProvider)

		// sys/registry resources
		registry.Provide(cfg, appDefBuilder, apis.IAppStructsProvider, apis.ITokens, apis.IFederation, ep)
		cfg.AddSyncProjectors(registry.ProvideSyncProjectorLoginIdxFactory())
		apps.RegisterSchemaFS(registrySchemaFS, RegistryAppFQN, ep)
	}
}

// Returns registry application definition
func AppDef() (appdef.IAppDef, error) {
	return parser.BuildAppDefFromFS(RegistryAppFQN, registrySchemaFS, "")
}

// Returns registry engines pool sizes
func Engines() [cluster.ProcessorKind_Count][]int {
	return [cluster.ProcessorKind_Count][]int{}
}

// Returns registry partitions count
func PartsCount() int { return 1 }
