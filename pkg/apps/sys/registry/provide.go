/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registryapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz/signupin"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(smtpCfg smtp.Cfg) apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {

		// sys package
		sys.Provide(cfg, appDefBuilder, smtpCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
			apis.NumCommandProcessors, nil, false, false)

		// sys/registry resources
		// note: q.sys.RefreshPrincipalToken is moved to sys package because it is strange to call it in sys/registry: provided token is issued for different app (e.g. airs-bp)
		signupin.Provide(cfg, appDefBuilder, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, ep)
		cfg.AddSyncProjectors(
			signupin.ProvideSyncProjectorLoginIdxFactory(),
		)
	}
}
