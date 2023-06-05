/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobberapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(smtpCfg smtp.Cfg) apps.AppBuilder {
	return func(appAPI apps.AppAPI, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
		sys.Provide(vvmCfg.TimeFunc, cfg, appDefBuilder, vvmAPI, smtpCfg, sep, nil, vvmCfg.NumCommandProcessors) // need to generate AppWorkspaces only
	}
}
