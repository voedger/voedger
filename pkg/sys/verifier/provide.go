/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package verifier

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func ProvideLimits(cfg *istructsmem.AppConfigType) {
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQueryInitiateEmailVerification, istructs.RateLimit{
		Period:                InitiateEmailVerification_Period,
		MaxAllowedPerDuration: InitiateEmailVerification_MaxAllowed,
	})

	// code ok -> buckets state will be reset
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQueryIssueVerifiedValueToken, RateLimit_IssueVerifiedValueToken)
}

func Provide(sr istructsmem.IStatelessResources, itokens itokens.ITokens, federation federation.IFederation, asp istructs.IAppStructsProvider,
	smtpCfg smtp.Cfg, timeFunc coreutils.TimeFunc) {
	provideQryInitiateEmailVerification(sr, itokens, asp, federation)
	provideQryIssueVerifiedValueToken(sr, itokens, asp)
	provideCmdSendEmailVerificationCode(sr)
	sr.AddProjectors(appdef.SysPackagePath,
		istructs.Projector{
			Name: qNameAPApplySendEmailVerificationCode,
			Func: applySendEmailVerificationCode(federation, smtpCfg, timeFunc),
		},
	)
}
