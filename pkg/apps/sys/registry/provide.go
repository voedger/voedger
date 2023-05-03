/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registryapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz/signupin"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/vvm"
)

func Provide(smtpCfg smtp.Cfg) vvm.HVMAppBuilder {
	return func(hvmCfg *vvm.HVMConfig, hvmAPI vvm.HVMAPI, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {

		// sys package
		sys.Provide(hvmCfg.TimeFunc, cfg, appDefBuilder, hvmAPI, smtpCfg, sep)

		// sys/registry resources
		// note: q.sys.RefreshPrincipalToken is moved to sys package because it is strange to call it in sys/registry: provided token is issued for different app (e.g. airs-bp)
		signupin.Provide(cfg, appDefBuilder, hvmAPI.ITokens, hvmAPI.FederationURL, hvmAPI.IAppStructsProvider)
		cfg.AddSyncProjectors(
			signupin.ProvideSyncProjectorLoginIdxFactory(),
		)
	}
}
