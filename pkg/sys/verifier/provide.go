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
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federation coreutils.IFederation, asp istructs.IAppStructsProvider) {
	provideQryInitiateEmailVerification(cfg, appDefBuilder, itokens, asp, federation)
	provideQryIssueVerifiedValueToken(cfg, appDefBuilder, itokens, asp)
	provideCmdSendEmailVerificationCode(cfg, appDefBuilder)
	appDefBuilder.AddStruct(qNameAPSendEmailVerificationCode, appdef.DefKind_Object)
}

func ProvideAsyncProjectorFactory_SendEmailVerificationCode(federation coreutils.IFederation, smtpCfg smtp.Cfg) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameAPSendEmailVerificationCode,
			Func:         sendEmailVerificationCodeProjector(federation, smtpCfg),
			EventsFilter: []appdef.QName{QNameCommandSendEmailVerificationCode},
		}
	}
}
