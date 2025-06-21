/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registry

import (
	"embed"
	"net/http"
	"regexp"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
)

const (
	RegistryPackage         = "registry"
	RegistryPackageFQN      = "github.com/voedger/voedger/pkg/" + RegistryPackage
	field_AppWSID           = "AppWSID"
	field_AppIDLoginHash    = "AppIDLoginHash"
	field_CDocLoginID       = "CDocLoginID"
	field_PwdHash           = "PwdHash"
	field_Passwrd           = "Password"
	field_NewPassword       = "NewPassword"
	field_OldPassword       = "OldPassword"
	field_Email             = "Email"
	field_Language          = "Language"
	field_VerificationToken = "VerificationToken"
	field_VerificationCode  = "VerificationCode"
	field_ProfileWSID       = "ProfileWSID"
	field_NewPwd            = "NewPwd"
	field_AppName           = "AppName"
	field_Login             = "Login"
	field_TTLHours          = "TTLHours"
	field_GlobalRoles       = "GlobalRoles"
	maxTokenTTLHours        = 168 // 1 week
)

var (
	validLoginRegexp                                  = regexp.MustCompile(`^[a-z0-9!#$%&'*+-\/=?^_{|}~@]+$`) // https://dev.untill.com/projects/#!537026
	QNameViewLoginIdx                                 = appdef.NewQName(RegistryPackage, "LoginIdx")
	qNameCmdChangePassword                            = appdef.NewQName(RegistryPackage, "ChangePassword")
	QNameProjectorLoginIdx                            = appdef.NewQName(RegistryPackage, "ProjectorLoginIdx")
	QNameCommandCreateLogin                           = appdef.NewQName(RegistryPackage, "CreateLogin")
	QNameCommandCreateEmailLogin                      = appdef.NewQName(RegistryPackage, "CreateEmailLogin")
	QNameCommandResetPasswordByEmail                  = appdef.NewQName(RegistryPackage, "ResetPasswordByEmail")
	QNameCommandUpdateGlobalRoles                     = appdef.NewQName(RegistryPackage, "UpdateGlobalRoles")
	QNameCommandResetPasswordByEmailUnloggedParams    = appdef.NewQName(RegistryPackage, "ResetPasswordByEmailUnloggedParams")
	QNameQueryInitiateResetPasswordByEmail            = appdef.NewQName(RegistryPackage, "InitiateResetPasswordByEmail")
	QNameQueryIssueVerifiedValueTokenForResetPassword = appdef.NewQName(RegistryPackage, "IssueVerifiedValueTokenForResetPassword")
	QNameCDocLogin                                    = appdef.NewQName(RegistryPackage, "Login")
	qNameProjectorInvokeCreateWorkspaceID_registry    = appdef.NewQName(RegistryPackage, "InvokeCreateWorkspaceID_registry")
	errPasswordIsIncorrect                            = coreutils.NewHTTPErrorf(http.StatusUnauthorized, "password is incorrect")
	errLoginOrPasswordIsIncorrect                     = coreutils.NewHTTPErrorf(http.StatusUnauthorized, "login or password is incorrect")

	//go:embed appws.vsql
	schemasFS embed.FS
)
