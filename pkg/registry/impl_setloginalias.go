/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package registry

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/jsonu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/uniques"
	"golang.org/x/crypto/bcrypt"
)

type federationCaller interface {
	Func(relativeURL string, body string, optFuncs ...httpu.ReqOptFunc) (*federation.FuncResponse, error)
}

type aliasOwner struct {
	sourceAppWSID istructs.WSID
	cdocLoginID   istructs.RecordID
}

type signInLogin struct {
	canonicalLogin string
	pwdHash        []byte
	profileWSID    int64
	wsError        string
	alias          string
	subjectKind    int32
	globalRoles    string
}

func execCmdInitiateSetLoginAlias(args istructs.ExecCommandArgs) error {
	login := args.ArgumentObject.AsString(field_Login)
	appName := args.ArgumentObject.AsString(field_AppName)
	alias := args.ArgumentObject.AsString(field_Alias)

	if err := CheckAppWSID(login, args.WSID, args.State.AppStructs().NumAppWorkspaces()); err != nil {
		return err
	}
	if len(alias) > 0 {
		if err := validateSignInIdentifier(alias); err != nil {
			return err
		}
	}

	cdocLogin, loginExists, err := GetCDocLogin(login, args.State, args.WSID, appName)
	if err != nil {
		return err
	}
	if !loginExists {
		return errLoginDoesNotExist(login)
	}
	if cdocLogin.AsInt32(field_AliasInProc) != 0 {
		return coreutils.NewHTTPErrorf(http.StatusConflict, "login alias update is already in progress")
	}

	kb, err := args.State.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return err
	}
	cdocLoginUpdater, err := args.Intents.UpdateValue(kb, cdocLogin)
	if err != nil {
		return err
	}
	cdocLoginUpdater.PutInt32(field_AliasInProc, 1)
	return nil
}

func applySetLoginAlias(fed federationCaller, tokens itokens.ITokens) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) error {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) error {
		login := event.ArgumentObject().AsString(field_Login)
		appName := event.ArgumentObject().AsString(field_AppName)
		newAlias := event.ArgumentObject().AsString(field_Alias)

		// Re-read the source Login in the command workspace. The initiating
		// command only marks AliasInProc; this projector owns the alias change.
		cdocLogin, loginExists, err := GetCDocLogin(login, s, event.Workspace(), appName)
		if err != nil {
			// No source CDoc ID has been resolved yet, so there is no reliable
			// Login record to update with AliasError. Return for projector retry.
			return err
		}
		if !loginExists {
			return errLoginDoesNotExist(login)
		}

		cdocLoginID := cdocLogin.AsRecordID(appdef.SystemField_ID)
		oldAlias := cdocLogin.AsString(field_Alias)
		if newAlias == oldAlias {
			// Idempotent retry or no-op request: clear transient state and keep
			// the current alias unchanged.
			return updateSourceLoginAliasViaCUD(fed, tokens, s.App(), event.Workspace(), cdocLoginID, oldAlias, "", 0)
		}

		if len(newAlias) > 0 {
			// Create the new alias index before committing Login.Alias. Sign-in
			// still verifies Login.Alias, so a partially created index is inert.
			targetAppWSID := coreutils.GetPseudoWSID(istructs.NullWSID, newAlias, event.Workspace().ClusterID())
			if err := callRegistryCommand(fed, tokens, s.App(), targetAppWSID, QNameCommandPutLoginAliasIndex, jsonu.Jprintf(
				`{"args":{"AppName":%q,"SourceAppWSID":%d,"Alias":%q,"CDocLoginID":%d,"Login":%q}}`,
				appName, event.Workspace(), newAlias, cdocLoginID, login)); err != nil {
				return updateSourceLoginAliasErrorViaCUD(fed, tokens, s.App(), event.Workspace(), cdocLoginID, err.Error())
			}
		}

		if len(oldAlias) > 0 {
			// Drop the old index before the source commit so old aliases stop
			// resolving as soon as the source Login moves to the new value.
			oldTargetAppWSID := coreutils.GetPseudoWSID(istructs.NullWSID, oldAlias, event.Workspace().ClusterID())
			if err := callRegistryCommand(fed, tokens, s.App(), oldTargetAppWSID, QNameCommandDeactivateLoginAliasIndex, jsonu.Jprintf(
				`{"args":{"AppName":%q,"SourceAppWSID":%d,"Alias":%q,"CDocLoginID":%d}}`,
				appName, event.Workspace(), oldAlias, cdocLoginID)); err != nil {
				return updateSourceLoginAliasErrorViaCUD(fed, tokens, s.App(), event.Workspace(), cdocLoginID, err.Error())
			}
		}

		// Commit the source Login last. This is the point where alias sign-in
		// becomes authoritative because resolution requires Login.Alias match.
		if err := updateSourceLoginAliasViaCUD(fed, tokens, s.App(), event.Workspace(), cdocLoginID, newAlias, "", 0); err != nil {
			bestEffortAliasErrorWrite(fed, tokens, s.App(), event.Workspace(), cdocLoginID, err)
			return err
		}
		return nil
	}
}

func execCmdPutLoginAliasIndex(args istructs.ExecCommandArgs) error {
	appName := args.ArgumentObject.AsString(field_AppName)
	alias := args.ArgumentObject.AsString(field_Alias)
	sourceAppWSID := istructs.WSID(args.ArgumentObject.AsInt64(field_SourceAppWSID)) // nolint G115
	cdocLoginID := istructs.RecordID(args.ArgumentObject.AsInt64(field_CDocLoginID)) // nolint G115

	if err := CheckAppWSID(alias, args.WSID, args.State.AppStructs().NumAppWorkspaces()); err != nil {
		return err
	}
	owner := aliasOwner{sourceAppWSID: sourceAppWSID, cdocLoginID: cdocLoginID}
	if err := assertIdentifierAvailable(args.State, appName, alias, args.WSID, &owner); err != nil {
		return err
	}

	existingAlias, err := getLoginAlias(args.State, args.WSID, appName, alias)
	if err != nil {
		return err
	}
	if existingAlias != nil {
		if existingAlias.AsBool(appdef.SystemField_IsActive) {
			return nil
		}
		return updateLoginAliasIndex(args, existingAlias, appName, sourceAppWSID, cdocLoginID, alias)
	}

	kb, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocLoginAlias)
	if err != nil {
		return err
	}
	loginAlias, err := args.Intents.NewValue(kb)
	if err != nil {
		return err
	}
	loginAlias.PutRecordID(appdef.SystemField_ID, 1)
	loginAlias.PutString(field_AppName, appName)
	loginAlias.PutInt64(field_SourceAppWSID, int64(sourceAppWSID)) // nolint G115: WSID highest bit is always 0
	loginAlias.PutInt64(field_CDocLoginID, int64(cdocLoginID))     // nolint G115: RecordID highest bit is always 0
	loginAlias.PutString(field_Login, args.ArgumentObject.AsString(field_Login))
	loginAlias.PutString(field_Alias, alias)
	return nil
}

func updateLoginAliasIndex(args istructs.ExecCommandArgs, loginAlias istructs.IStateValue, appName string, sourceAppWSID istructs.WSID, cdocLoginID istructs.RecordID, alias string) error {
	kb, err := args.State.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return err
	}
	loginAliasUpdater, err := args.Intents.UpdateValue(kb, loginAlias)
	if err != nil {
		return err
	}
	loginAliasUpdater.PutBool(appdef.SystemField_IsActive, true)
	loginAliasUpdater.PutString(field_AppName, appName)
	loginAliasUpdater.PutInt64(field_SourceAppWSID, int64(sourceAppWSID)) // nolint G115: WSID highest bit is always 0
	loginAliasUpdater.PutInt64(field_CDocLoginID, int64(cdocLoginID))     // nolint G115: RecordID highest bit is always 0
	loginAliasUpdater.PutString(field_Login, args.ArgumentObject.AsString(field_Login))
	loginAliasUpdater.PutString(field_Alias, alias)
	return nil
}

func execCmdDeactivateLoginAliasIndex(args istructs.ExecCommandArgs) error {
	appName := args.ArgumentObject.AsString(field_AppName)
	alias := args.ArgumentObject.AsString(field_Alias)
	sourceAppWSID := istructs.WSID(args.ArgumentObject.AsInt64(field_SourceAppWSID)) // nolint G115
	cdocLoginID := istructs.RecordID(args.ArgumentObject.AsInt64(field_CDocLoginID)) // nolint G115

	if err := CheckAppWSID(alias, args.WSID, args.State.AppStructs().NumAppWorkspaces()); err != nil {
		return err
	}

	loginAlias, err := getActiveLoginAlias(args.State, args.WSID, appName, alias)
	if err != nil || loginAlias == nil {
		return err
	}

	// nolint G115
	if loginAlias.AsInt64(field_SourceAppWSID) != int64(sourceAppWSID) ||
		loginAlias.AsInt64(field_CDocLoginID) != int64(cdocLoginID) {
		return coreutils.NewHTTPErrorf(http.StatusConflict, "login alias belongs to another login")
	}

	kb, err := args.State.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return err
	}
	loginAliasUpdater, err := args.Intents.UpdateValue(kb, loginAlias)
	if err != nil {
		return err
	}
	loginAliasUpdater.PutBool(appdef.SystemField_IsActive, false)
	return nil
}

func validateSignInIdentifier(identifier string) error {
	if strings.HasPrefix(identifier, "-") || strings.HasPrefix(identifier, ".") || strings.HasPrefix(identifier, " ") ||
		strings.HasSuffix(identifier, "-") || strings.HasSuffix(identifier, ".") || strings.HasSuffix(identifier, " ") ||
		strings.Contains(identifier, "..") || strings.HasPrefix(identifier, "sys.") || !validLoginRegexp.MatchString(identifier) {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, "incorrect login format: ", identifier)
	}
	return nil
}

func assertIdentifierAvailable(st istructs.IState, appName, identifier string, targetAppWSID istructs.WSID, allowedAliasOwner *aliasOwner) error {
	_, loginExists, err := GetCDocLogin(identifier, st, targetAppWSID, appName)
	if err != nil {
		return err
	}
	if loginExists {
		return coreutils.NewHTTPErrorf(http.StatusConflict, "sign-in identifier already exists")
	}

	loginAlias, err := getActiveLoginAlias(st, targetAppWSID, appName, identifier)
	if err != nil || loginAlias == nil {
		return err
	}
	// nolint G115
	if allowedAliasOwner != nil &&
		loginAlias.AsInt64(field_SourceAppWSID) == int64(allowedAliasOwner.sourceAppWSID) &&
		loginAlias.AsInt64(field_CDocLoginID) == int64(allowedAliasOwner.cdocLoginID) {
		return nil
	}
	return coreutils.NewHTTPErrorf(http.StatusConflict, "sign-in identifier already exists")
}

func getActiveLoginAlias(st istructs.IState, wsid istructs.WSID, appName, alias string) (istructs.IStateValue, error) {
	loginAlias, err := getLoginAlias(st, wsid, appName, alias)
	if err != nil || loginAlias == nil {
		return nil, err
	}
	if !loginAlias.AsBool(appdef.SystemField_IsActive) {
		return nil, nil
	}
	return loginAlias, nil
}

func getLoginAlias(st istructs.IState, wsid istructs.WSID, appName, alias string) (istructs.IStateValue, error) {
	recordID, err := uniques.GetRecordIDByUniqueCombination(wsid, QNameCDocLoginAlias, st.AppStructs(), map[string]interface{}{
		field_AppName: appName,
		field_Alias:   alias,
	})
	if err != nil || recordID == istructs.NullRecordID {
		return nil, err
	}

	kb, err := st.KeyBuilder(sys.Storage_Record, QNameCDocLoginAlias)
	if err != nil {
		return nil, err
	}
	kb.PutRecordID(sys.Storage_Record_Field_ID, recordID)
	loginAlias, err := st.MustExist(kb)
	if err != nil {
		return nil, err
	}
	return loginAlias, nil
}

func callRegistryFunc(fed federationCaller, tokens itokens.ITokens, appQName appdef.AppQName, wsid istructs.WSID, resource string, body string, optFuncs ...httpu.ReqOptFunc) (*federation.FuncResponse, error) {
	token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
	if err != nil {
		return nil, err
	}
	opts := append([]httpu.ReqOptFunc{httpu.WithAuthorizeBy(token)}, optFuncs...)
	return fed.Func(fmt.Sprintf("api/%s/%d/%s", appQName, wsid, resource), body, opts...)
}

func callRegistryCommand(fed federationCaller, tokens itokens.ITokens, appQName appdef.AppQName, wsid istructs.WSID, command appdef.QName, body string) error {
	_, err := callRegistryFunc(fed, tokens, appQName, wsid, fmt.Sprintf("c.%s", command), body, httpu.WithDiscardResponse())
	return err
}

func updateSourceLoginAliasViaCUD(fed federationCaller, tokens itokens.ITokens, appQName appdef.AppQName, wsid istructs.WSID, cdocLoginID istructs.RecordID, alias, aliasError string, aliasInProc int32) error {
	body := jsonu.Jprintf(`{"cuds":[{"sys.ID":%d,"fields":{%q:%q,%q:%q,%q:%d}}]}`,
		cdocLoginID, field_Alias, alias, field_AliasError, aliasError, field_AliasInProc, aliasInProc)
	_, err := callRegistryFunc(fed, tokens, appQName, wsid, "c.sys.CUD", body, httpu.WithDiscardResponse())
	return err
}

func bestEffortAliasErrorWrite(fed federationCaller, tokens itokens.ITokens, appQName appdef.AppQName, wsid istructs.WSID, cdocLoginID istructs.RecordID, sourceErr error) {
	_ = updateSourceLoginAliasErrorViaCUD(fed, tokens, appQName, wsid, cdocLoginID, sourceErr.Error())
}

func updateSourceLoginAliasErrorViaCUD(fed federationCaller, tokens itokens.ITokens, appQName appdef.AppQName, wsid istructs.WSID, cdocLoginID istructs.RecordID, aliasError string) error {
	body := jsonu.Jprintf(`{"cuds":[{"sys.ID":%d,"fields":{%q:%q,%q:0}}]}`,
		cdocLoginID, field_AliasError, aliasError, field_AliasInProc)
	_, err := callRegistryFunc(fed, tokens, appQName, wsid, "c.sys.CUD", body, httpu.WithDiscardResponse())
	return err
}

func loginFromPrimaryCDoc(login string, cdocLogin istructs.IStateValue) signInLogin {
	return signInLogin{
		canonicalLogin: login,
		pwdHash:        cdocLogin.AsBytes(field_PwdHash),
		profileWSID:    cdocLogin.AsInt64(authnz.Field_WSID),
		wsError:        cdocLogin.AsString(authnz.Field_WSError),
		alias:          cdocLogin.AsString(field_Alias),
		subjectKind:    cdocLogin.AsInt32(authnz.Field_SubjectKind),
		globalRoles:    cdocLogin.AsString(authnz.Field_GlobalRoles),
	}
}

func resolveAliasSignInLogin(submittedLogin, appName string, st istructs.IState, wsid istructs.WSID, tokens itokens.ITokens, fed federationCaller) (signInLogin, bool, error) {
	loginAlias, err := getActiveLoginAlias(st, wsid, appName, submittedLogin)
	if err != nil || loginAlias == nil {
		return signInLogin{}, false, err
	}

	sourceAppWSID := istructs.WSID(loginAlias.AsInt64(field_SourceAppWSID)) // nolint G115
	cdocLoginID := loginAlias.AsInt64(field_CDocLoginID)
	sourceLoginMap, err := getCDocViaFederation(fed, tokens, st.App(), sourceAppWSID, cdocLoginID)
	if err != nil {
		return signInLogin{}, false, err
	}
	if active, ok := sourceLoginMap[appdef.SystemField_IsActive].(bool); ok && !active {
		return signInLogin{}, false, nil
	}
	if str(sourceLoginMap[field_Alias]) != submittedLogin {
		return signInLogin{}, false, nil
	}

	pwdHash, err := bytesFromJSON(sourceLoginMap[field_PwdHash])
	if err != nil {
		return signInLogin{}, false, err
	}
	profileWSID, err := int64FromJSON(sourceLoginMap[authnz.Field_WSID])
	if err != nil {
		return signInLogin{}, false, err
	}
	subjectKind, err := int32FromJSON(sourceLoginMap[authnz.Field_SubjectKind])
	if err != nil {
		return signInLogin{}, false, err
	}
	return signInLogin{
		canonicalLogin: loginAlias.AsString(field_Login),
		pwdHash:        pwdHash,
		profileWSID:    profileWSID,
		wsError:        str(sourceLoginMap[authnz.Field_WSError]),
		alias:          str(sourceLoginMap[field_Alias]),
		subjectKind:    subjectKind,
		globalRoles:    str(sourceLoginMap[authnz.Field_GlobalRoles]),
	}, true, nil
}

func getCDocViaFederation(fed federationCaller, tokens itokens.ITokens, appQName appdef.AppQName, wsid istructs.WSID, id int64) (map[string]interface{}, error) {
	resp, err := callRegistryFunc(fed, tokens, appQName, wsid, "q.sys.GetCDoc",
		fmt.Sprintf(`{"args":{"ID":%d},"elements":[{"fields":["Result"]}]}`, id))
	if err != nil {
		return nil, err
	}
	login := map[string]interface{}{}
	if err := coreutils.JSONUnmarshal([]byte(resp.SectionRow()[0].(string)), &login); err != nil {
		return nil, err
	}
	return login, nil
}

func checkPasswordHash(pwdHash []byte, pwd string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(pwdHash, []byte(pwd)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("failed to authenticate: %w", err)
	}
	return true, nil
}

func str(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func bytesFromJSON(v interface{}) ([]byte, error) {
	switch typed := v.(type) {
	case []byte:
		return typed, nil
	case string:
		return base64.StdEncoding.DecodeString(typed)
	default:
		return nil, fmt.Errorf("unexpected bytes JSON value %T", v)
	}
}

func int64FromJSON(v interface{}) (int64, error) {
	switch typed := v.(type) {
	case json.Number:
		return typed.Int64()
	case float64:
		return int64(typed), nil
	case int64:
		return typed, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unexpected int64 JSON value %T", v)
	}
}

func int32FromJSON(v interface{}) (int32, error) {
	val, err := int64FromJSON(v)
	if err != nil {
		return 0, err
	}
	return int32(val), nil // nolint G115: schema stores int32
}
