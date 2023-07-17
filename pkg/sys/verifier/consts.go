/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package verifier

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	Field_Email              = "Email"
	field_Entity             = "Entity"
	field_Field              = "Field"
	field_VerificationToken  = "VerificationToken"
	field_VerificationCode   = "VerificationCode"
	field_VerifiedValueToken = "VerifiedValueToken"
	field_Reason             = "Reason"
	field_TargetWSID         = "TargetWSID"
	field_ForRegistry        = "ForRegistry"
	field_Language           = "Language"

	VerifiedValueTokenDuration              = 10 * time.Minute
	VerificationTokenDuration               = 10 * time.Minute
	VerificationCodeLength                  = 6
	verificationCodeSymbols                 = "1234567890"
	maxByte                                 = ^byte(0)
	byteRangeToVerifcationSymbolsRangeCoeff = (float32(maxByte) + 1) / float32(len(verificationCodeSymbols))
	EmailSubject                            = "Your verification code"
	EmailFrom                               = "noreply@air.com"
	InitiateEmailVerification_Period        = time.Hour
	InitiateEmailVerification_MaxAllowed    = uint32(3)
	IssueVerifiedValueToken_Period          = time.Hour
	IssueVerifiedValueToken_MaxAllowed      = uint32(3)
	verifyEmailReason                       = "confirm your E-mail"
)

var (
	QNameCommandSendEmailVerificationCode = appdef.NewQName(appdef.SysPackage, "SendEmailVerificationCode")
	QNameQueryInitiateEmailVerification   = appdef.NewQName(appdef.SysPackage, "InitiateEmailVerification")
	QNameQueryIssueVerifiedValueToken     = appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueToken")
	RateLimit_IssueVerifiedValueToken     = istructs.RateLimit{
		Period:                IssueVerifiedValueToken_Period,
		MaxAllowedPerDuration: IssueVerifiedValueToken_MaxAllowed,
	}
	qNameAPSendEmailVerificationCode = appdef.NewQName(appdef.SysPackage, "SendEmailVerificationCode")
)
