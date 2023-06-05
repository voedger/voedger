/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package routerapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(smtpCfg smtp.Cfg) apps.AppBuilder {
	return func(appAPI apps.AppAPI, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
		sys.Provide(appAPI, cfg, appDefBuilder, smtpCfg, ep, nil, appAPI.IAppStructsProvider, appAPI.ITokens, appAPI.IFederation, appAPI.IAppTokensFactory, vvmCfg.NumCommandProcessors) // need to generate AppWorkspaces only
	}
}
