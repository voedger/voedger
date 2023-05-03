/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package routerapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/vvm"
)

func Provide(smtpCfg smtp.Cfg) vvm.HVMAppBuilder {
	return func(hvmCfg *vvm.HVMConfig, hvmAPI vvm.HVMAPI, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {
		sys.Provide(hvmCfg.TimeFunc, cfg, appDefBuilder, hvmAPI, smtpCfg, sep) // need to generate AppWorkspaces only
	}
}
