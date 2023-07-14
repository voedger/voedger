/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package signupin

import (
	"regexp"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

const (
	Field_AppName                   = "AppName"
	field_Password                  = "Password"
	field_AppWSID                   = "AppWSID"
	field_AppIDLoginHash            = "AppIDLoginHash"
	field_Passwrd                   = "Password"
	field_PwdHash                   = "PwdHash"
	field_CDocLoginID               = "CDocLoginID"
	field_NewPwd                    = "NewPwd"
	field_Email                     = "Email"
	field_VerificationToken         = "VerificationToken"
	field_VerificationCode          = "VerificationCode"
	field_VerifiedValueToken        = "VerifiedValueToken"
	field_ProfileWSID               = "ProfileWSID"
	field_ExistingPrincipalToken    = "ExistingPrincipalToken"
	field_NewPrincipalToken         = "NewPrincipalToken"
	field_EnrichedToken             = "EnrichedToken"
	field_Login                     = "Login"
	field_NewPassword               = "NewPassword"
	field_OldPassword               = "OldPassword"
	field_Language                  = "Language"
	DefaultPrincipalTokenExpiration = 24 * time.Hour
)

var (
	// see https://dev.untill.com/projects/#!537026
	validLoginRegexp       *regexp.Regexp = regexp.MustCompile(`^[a-z0-9!#$%&'*+-\/=?^_{|}~@]+$`)
	QNameViewLoginIdx                     = appdef.NewQName(appdef.SysPackage, "LoginIdx")
	qNameCmdChangePassword                = appdef.NewQName(appdef.SysPackage, "ChangePassword")
)

const (
	ErrMessageLoginOrPasswordIsIncorrect = "login or password is incorrect"
	ErrMessagePasswordIsIncorrect        = "password is incorrect"
	ErrFormatMessageLoginDoesntExist     = "login %s does not exist"
)
