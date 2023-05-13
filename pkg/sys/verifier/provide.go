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
	"github.com/voedger/voedger/pkg/vvm"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federationURL vvm.FederationURLType, asp istructs.IAppStructsProvider) {
	provideQryInitiateEmailVerification(cfg, appDefBuilder, itokens, asp, federationURL)
	provideQryIssueVerifiedValueToken(cfg, appDefBuilder, itokens, asp)
	provideCmdSendEmailVerificationCode(cfg, appDefBuilder)
	appDefBuilder.AddStruct(qNameAPSendEmailVerificationCode, appdef.DefKind_Object)
}

func ProvideAsyncProjectorFactory_SendEmailVerificationCode(federationURL vvm.FederationURLType, smtpCfg smtp.Cfg) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameAPSendEmailVerificationCode,
			Func:         sendEmailVerificationCodeProjector(federationURL, smtpCfg),
			EventsFilter: []appdef.QName{QNameCommandSendEmailVerificationCode},
		}
	}
}
