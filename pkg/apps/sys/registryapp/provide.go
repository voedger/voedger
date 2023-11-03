/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registryapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/registry"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(smtpCfg smtp.Cfg, rebuildRegistry bool) apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {

		// sys package
		sys.Provide(cfg, appDefBuilder, smtpCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
			apis.NumCommandProcessors, nil, apis.IAppStorageProvider, rebuildRegistry)

		// sys/registry resources
		registry.Provide(cfg, appDefBuilder, apis.IAppStructsProvider, apis.ITokens, apis.IFederation, ep)
		cfg.AddSyncProjectors(registry.ProvideSyncProjectorLoginIdxFactory())
		apps.RegisterSchemaFS(registrySchemaFS, "github.com/voedger/voedger/pkg/apps/sys/regfistryapp", ep)
	}
}
