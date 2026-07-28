/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/jsonu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/verifier"
)

const field_CanonicalPseudoWSID = "CanonicalPseudoWSID"

type resetPasswordLogin struct {
	profileWSID         int64
	canonicalPseudoWSID int64
}

type resetPasswordResult struct {
	istructs.NullObject
	token               string
	profileWSID         int64
	canonicalPseudoWSID int64
}

func provideResetPassword(cfgRegistry *istructsmem.AppConfigType, itokens itokens.ITokens, federation federation.IFederation) {
	// sys/registry/pseudoProfileWSID/q.sys.InitiateResetPasswordByEmail
	// null auth
	cfgRegistry.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryInitiateResetPasswordByEmail,
		provideQryInitiateResetPasswordByEmailExec(itokens, federation),
	))

	// sys/registry/pseudoProfileWSID/q.registry.IssueVerifiedValueTokenForResetPassword
	// null auth
	cfgRegistry.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryIssueVerifiedValueTokenForResetPassword,
		provideIssueVerifiedValueTokenForResetPasswordExec(itokens, federation),
	))

	cfgRegistry.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandResetPasswordByEmail,
		cmdResetPasswordByEmailExec,
	))
}

// sys/registry/pseudoWSID
// null auth
func provideQryInitiateResetPasswordByEmailExec(itokens itokens.ITokens, federation federation.IFederation) istructsmem.ExecQueryClosure {
	return func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		loginAppStr := args.ArgumentObject.AsString(authnz.Field_AppName)
		email := args.ArgumentObject.AsString(field_Email)
		language := args.ArgumentObject.AsString(field_Language)

		loginAppQName, err := appdef.ParseAppQName(loginAppStr)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		loginForReset, err := resolveResetPasswordLogin(email, loginAppStr, args.State, args.WSID, itokens, federation)
		if err != nil {
			return err
		}

		// targetWSID - is the workspace we're going to use the verified value at
		body := jsonu.Jprintf(`{"args":{"Entity":%q,"Field":%q,"Email":%q,"TargetWSID":%d,"ForRegistry":true,"Language":%q},"elements":[{"fields":["VerificationToken"]}]}`,
			QNameCommandResetPasswordByEmailUnloggedParams, field_Email, email, loginForReset.profileWSID, language)
		resp, err := callRegistryFunc(federation, itokens, loginAppQName, istructs.WSID(loginForReset.profileWSID), "q.sys.InitiateEmailVerification", body)
		if err != nil {
			return fmt.Errorf("q.sys.InitiateEmailVerification failed: %w", err)
		}

		verificationToken := resp.SectionRow()[0].(string)
		return callback(&resetPasswordResult{
			token:               verificationToken,
			profileWSID:         loginForReset.profileWSID,
			canonicalPseudoWSID: loginForReset.canonicalPseudoWSID,
		})
	}
}

// sys/registry/pseudoWSID
// null auth
func provideIssueVerifiedValueTokenForResetPasswordExec(itokens itokens.ITokens, federation federation.IFederation) istructsmem.ExecQueryClosure {
	return func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		token := args.ArgumentObject.AsString(field_VerificationToken)
		code := args.ArgumentObject.AsString(field_VerificationCode)
		profileWSID := args.ArgumentObject.AsInt64(field_ProfileWSID)
		loginAppStr := args.ArgumentObject.AsString(authnz.Field_AppName)

		loginAppQName, err := appdef.ParseAppQName(loginAppStr)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		body := jsonu.Jprintf(`{"args":{"VerificationToken":%q,"VerificationCode":%q,"ForRegistry":true},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code)
		resp, err := callRegistryFunc(federation, itokens, loginAppQName, istructs.WSID(profileWSID), "q.sys.IssueVerifiedValueToken", body)
		if err != nil {
			return err
		}
		verificationToken := resp.SectionRow()[0].(string)
		verificationToken, err = reissueResetPasswordTokenForAlias(verificationToken, loginAppStr, args.State, args.WSID)
		if err != nil {
			return err
		}
		return callback(&resetPasswordResult{token: verificationToken})
	}
}

// sys/registry/pseudoWSID
// null auth
func cmdResetPasswordByEmailExec(args istructs.ExecCommandArgs) (err error) {
	email := args.ArgumentUnloggedObject.AsString(field_Email)
	newPwd := args.ArgumentUnloggedObject.AsString(field_NewPwd)
	appName := args.ArgumentObject.AsString(authnz.Field_AppName)
	login := email

	return ChangePassword(login, args.State, args.Intents, args.WSID, appName, newPwd)
}

func resolveResetPasswordLogin(email, appName string, st istructs.IState, wsid istructs.WSID, tokens itokens.ITokens, fed federationCaller) (resetPasswordLogin, error) {
	cdocLogin, loginExists, err := GetCDocLogin(email, st, wsid, appName)
	if err != nil {
		return resetPasswordLogin{}, err
	}
	if loginExists {
		profileWSID := cdocLogin.AsInt64(authnz.Field_WSID)
		if err := ensureResetPasswordProfileReady(profileWSID); err != nil {
			return resetPasswordLogin{}, err
		}
		return resetPasswordLogin{
			profileWSID:         profileWSID,
			canonicalPseudoWSID: int64(wsid),
		}, nil
	}

	loginAlias, err := getActiveLoginAlias(st, wsid, appName, email)
	if err != nil {
		return resetPasswordLogin{}, err
	}
	if loginAlias == nil {
		return resetPasswordLogin{}, errResetPasswordLoginDoesNotExist()
	}

	sourceAppWSID := istructs.WSID(loginAlias.AsInt64(field_SourceAppWSID)) // nolint G115
	sourceLoginMap, err := getCDocViaFederation(fed, tokens, st.App(), sourceAppWSID, loginAlias.AsInt64(field_CDocLoginID))
	if err != nil {
		return resetPasswordLogin{}, err
	}
	if active, ok := sourceLoginMap[appdef.SystemField_IsActive].(bool); ok && !active {
		return resetPasswordLogin{}, errResetPasswordLoginDoesNotExist()
	}
	if str(sourceLoginMap[field_Alias]) != email {
		return resetPasswordLogin{}, errResetPasswordLoginDoesNotExist()
	}

	profileWSID, err := int64FromJSON(sourceLoginMap[authnz.Field_WSID])
	if err != nil {
		return resetPasswordLogin{}, err
	}
	if err := ensureResetPasswordProfileReady(profileWSID); err != nil {
		return resetPasswordLogin{}, err
	}

	canonicalLogin := loginAlias.AsString(field_Login)
	canonicalPseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, canonicalLogin, wsid.ClusterID())
	return resetPasswordLogin{
		profileWSID:         profileWSID,
		canonicalPseudoWSID: int64(canonicalPseudoWSID),
	}, nil
}

func reissueResetPasswordTokenForAlias(token, appName string, st istructs.IState, wsid istructs.WSID) (string, error) {
	payload := payloads.VerifiedValuePayload{}
	if _, err := st.AppStructs().AppTokens().ValidateToken(token, &payload); err != nil {
		return "", err
	}

	email := str(payload.Value)
	if email == "" {
		return token, nil
	}
	loginAlias, err := getActiveLoginAlias(st, wsid, appName, email)
	if err != nil || loginAlias == nil {
		return token, err
	}

	payload.Value = loginAlias.AsString(field_Login)
	return st.AppStructs().AppTokens().IssueToken(verifier.VerifiedValueTokenDuration, &payload)
}

func errResetPasswordLoginDoesNotExist() error {
	return coreutils.NewHTTPErrorf(http.StatusBadRequest, "login does not exist")
}

func ensureResetPasswordProfileReady(profileWSID int64) error {
	if profileWSID == 0 {
		return coreutils.NewHTTPErrorf(http.StatusLocked, "login profile is not initialized")
	}
	return nil
}

func (r *resetPasswordResult) AsString(string) string {
	return r.token
}

func (r *resetPasswordResult) AsInt64(field string) int64 {
	switch field {
	case field_ProfileWSID:
		return r.profileWSID
	case field_CanonicalPseudoWSID:
		return r.canonicalPseudoWSID
	default:
		return 0
	}
}
