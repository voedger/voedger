/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package verifier

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(sr istructsmem.IStatelessResources, itokens itokens.ITokens, federation federation.IFederation, asp istructs.IAppStructsProvider,
	smtpCfg smtp.Cfg) {
	provideQryInitiateEmailVerification(sr, itokens, asp, federation)
	provideQryIssueVerifiedValueToken(sr, itokens, asp)
	provideCmdSendEmailVerificationCode(sr)
	sr.AddProjectors(appdef.SysPackagePath,
		istructs.Projector{
			Name: qNameAPApplySendEmailVerificationCode,
			Func: applySendEmailVerificationCode(federation, smtpCfg),
		},
	)
}
