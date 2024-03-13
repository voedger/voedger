/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package verifier

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, itokens itokens.ITokens, federation coreutils.IFederation, asp istructs.IAppStructsProvider,
	smtpCfg smtp.Cfg, timeFunc coreutils.TimeFunc) {
	provideQryInitiateEmailVerification(cfg, itokens, asp, federation)
	provideQryIssueVerifiedValueToken(cfg, itokens, asp)
	provideCmdSendEmailVerificationCode(cfg)
	cfg.AddAsyncProjectors(func() istructs.Projector {
		return istructs.Projector{
			Name: qNameAPApplySendEmailVerificationCode,
			Func: applySendEmailVerificationCode(federation, smtpCfg, timeFunc),
		}
	})
}
