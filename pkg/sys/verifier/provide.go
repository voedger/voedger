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

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federation coreutils.IFederation, asp istructs.IAppStructsProvider,
	smtpCfg smtp.Cfg) {
	provideQryInitiateEmailVerification(cfg, appDefBuilder, itokens, asp, federation)
	provideQryIssueVerifiedValueToken(cfg, appDefBuilder, itokens, asp)
	provideCmdSendEmailVerificationCode(cfg, appDefBuilder)
	appDefBuilder.AddObject(qNameAPSendEmailVerificationCodeProjector)
	cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameAPSendEmailVerificationCodeProjector,
			Func:         sendEmailVerificationCodeProjector(federation, smtpCfg),
			EventsFilter: []appdef.QName{QNameCommandSendEmailVerificationCode},
			NonBuffered:  true,
		}
	})
}
