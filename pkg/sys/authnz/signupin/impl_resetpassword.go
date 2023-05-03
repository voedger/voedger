/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package signupin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vvm"
)

func provideResetPassword(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, federationURL vvm.FederationURLType,
	asp istructs.IAppStructsProvider) {

	// sys/registry/pseudoProfileWSID/q.sys.InitiateResetPasswordByEmail
	// null auth
	cfgRegistry.Resources.Add(istructsmem.NewQueryFunction(
		authnz.QNameQueryInitiateResetPasswordByEmail,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "InitiateResetPasswordByEmailParams"), appdef.DefKind_Object).
			AddField(Field_AppName, appdef.DataKind_string, true).
			AddField(field_Email, appdef.DataKind_string, true).QName(),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "InitiateResetPasswordByEmailResult"), appdef.DefKind_Object).
			AddField(field_VerificationToken, appdef.DataKind_string, true).
			AddField(field_ProfileWSID, appdef.DataKind_int64, true).
			QName(),
		provideQryInitiateResetPasswordByEmailExec(itokens, federationURL, asp),
	))

	// sys/registry/pseudoProfileWSID/q.sys.IssueVerifiedValueTokenForResetPassword
	// null auth
	cfgRegistry.Resources.Add(istructsmem.NewQueryFunction(
		authnz.QNameQueryIssueVerifiedValueTokenForResetPassword,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueTokenForResetPasswordParams"), appdef.DefKind_Object).
			AddField(field_VerificationToken, appdef.DataKind_string, true).
			AddField(field_VerificationCode, appdef.DataKind_string, true).
			AddField(field_ProfileWSID, appdef.DataKind_int64, true).
			AddField(Field_AppName, appdef.DataKind_string, true).QName(),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueTokenForResetPasswordResult"), appdef.DefKind_Object).
			AddField(field_VerifiedValueToken, appdef.DataKind_string, true).QName(),
		provideIssueVerifiedValueTokenForResetPasswordExec(itokens, federationURL),
	))

	cfgRegistry.Resources.Add(istructsmem.NewCommandFunction(
		authnz.QNameCommandResetPasswordByEmail,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "ResetPasswordByEmailParams"), appdef.DefKind_Object).
			AddField(Field_AppName, appdef.DataKind_string, true).QName(),
		appDefBuilder.AddStruct(authnz.QNameCommandResetPasswordByEmailUnloggedParams, appdef.DefKind_Object).
			AddVerifiedField(field_Email, appdef.DataKind_string, true, appdef.VerificationKind_EMail).
			AddField(field_NewPwd, appdef.DataKind_string, true).QName(),
		appdef.NullQName,
		cmdResetPasswordByEmailExec,
	))
}

// sys/registry/pseudoWSID
// null auth
func provideQryInitiateResetPasswordByEmailExec(itokens itokens.ITokens, federationUrl vvm.FederationURLType, asp istructs.IAppStructsProvider) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		loginAppStr := args.ArgumentObject.AsString(Field_AppName)
		email := args.ArgumentObject.AsString(field_Email)
		login := email // TODO: considering login is email

		loginAppQName, err := istructs.ParseAppQName(loginAppStr)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		as, err := asp.AppStructs(loginAppQName)
		if err != nil {
			return err
		}

		// request is sent to pseudoProfileWSID, translated to AppWS
		if err = CheckAppWSID(login, args.Workspace, as.WSAmount()); err != nil {
			return err
		}

		cdocLoginID, err := GetCDocLoginID(args.State, args.Workspace, loginAppStr, login)
		if err != nil {
			return err
		}
		if cdocLoginID == 0 {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "login does not exist")
		}

		// check CDoc<sys.Login>.WSID != 0
		kb, err := args.State.KeyBuilder(state.RecordsStorage, authnz.QNameCDocLogin)
		if err != nil {
			return err
		}
		kb.PutRecordID(state.Field_ID, cdocLoginID)
		sv, err := args.State.MustExist(kb)
		if err != nil {
			return err
		}
		profileWSID := sv.AsInt64(authnz.Field_WSID)
		if profileWSID == 0 {
			return coreutils.NewHTTPErrorf(http.StatusLocked, "login profile is not initialized")
		}

		sysToken, err := payloads.GetSystemPrincipalToken(itokens, loginAppQName)
		if err != nil {
			return err
		}
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"%s","Email":"%s","TargetWSID":%d,"ForRegistry":true},"elements":[{"fields":["VerificationToken"]}]}`,
			authnz.QNameCommandResetPasswordByEmailUnloggedParams, field_Email, email, profileWSID) // targetWSID - is the workspace we're going to use the verified value at
		resp, err := utils.FederationFunc(federationUrl(), fmt.Sprintf("api/%s/%d/q.sys.InitiateEmailVerification", loginAppQName, profileWSID), body, coreutils.WithAuthorizeBy(sysToken))
		if err != nil {
			return fmt.Errorf("q.sys.InitiateEmailVerification failed: %w", err)
		}

		verificationToken := resp.SectionRow()[0].(string)
		return callback(&result{token: verificationToken, profileWSID: profileWSID})
	}
}

// sys/registry/pseudoWSID
// null auth
func provideIssueVerifiedValueTokenForResetPasswordExec(itokens itokens.ITokens, federationUrl vvm.FederationURLType) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		token := args.ArgumentObject.AsString(field_VerificationToken)
		code := args.ArgumentObject.AsString(field_VerificationCode)
		profileWSID := args.ArgumentObject.AsInt64(field_ProfileWSID)
		loginAppStr := args.ArgumentObject.AsString(Field_AppName)

		loginAppQName, err := istructs.ParseAppQName(loginAppStr)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		sysToken, err := payloads.GetSystemPrincipalToken(itokens, loginAppQName)
		if err != nil {
			return err
		}

		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ForRegistry":true},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code)
		resp, err := utils.FederationFunc(federationUrl(), fmt.Sprintf("api/%s/%d/q.sys.IssueVerifiedValueToken", loginAppQName, profileWSID), body, coreutils.WithAuthorizeBy(sysToken))
		if err != nil {
			return err
		}
		verificationToken := resp.SectionRow()[0].(string)
		return callback(&result{token: verificationToken})
	}
}

// sys/registry/pseudoWSID
// null auth
func cmdResetPasswordByEmailExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	email := args.ArgumentUnloggedObject.AsString(field_Email)
	newPwd := args.ArgumentUnloggedObject.AsString(field_NewPwd)
	appName := args.ArgumentObject.AsString(Field_AppName)
	login := email

	return ChangePassword(login, args.State, args.Intents, args.Workspace, appName, newPwd)
}

func (r *result) AsString(string) string {
	return r.token
}

func (r *result) AsInt64(string) int64 {
	return r.profileWSID
}
