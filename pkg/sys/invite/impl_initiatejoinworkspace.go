/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func provideCmdInitiateJoinWorkspace(sr istructsmem.IStatelessResources, time timeu.ITime, tokens itokens.ITokens) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateJoinWorkspace,
		execCmdInitiateJoinWorkspace(time, tokens),
	))
}

// [~server.invites/Join.InitiateJoinWorkspace~impl]
func execCmdInitiateJoinWorkspace(tm timeu.ITime, tokens itokens.ITokens) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		inviteID := args.ArgumentObject.AsRecordID(field_InviteID)
		skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, inviteID)
		svCDocInvite, ok, err := args.State.CanExist(skbCDocInvite)
		if err != nil {
			return
		}
		if !ok {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteNotExists)
		}

		if !isValidInviteState(svCDocInvite.AsInt32(Field_State), qNameCmdInitiateJoinWorkspace) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}
		if svCDocInvite.AsInt64(field_ExpireDatetime) < tm.Now().UnixMilli() {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteExpired)
		}
		if svCDocInvite.AsString(field_VerificationCode) != args.ArgumentObject.AsString(field_VerificationCode) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteVerificationCodeInvalid)
		}

		skbPrincipal, err := args.State.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
		if err != nil {
			return
		}
		svPrincipal, err := args.State.MustExist(skbPrincipal)
		if err != nil {
			return
		}

		principalPayload := payloads.PrincipalPayload{}
		if _, err = payloads.GetPayloadRegistry(tokens, svPrincipal.AsString(sys.Storage_RequestSubject_Field_Token), &principalPayload); err != nil {
			return
		}

		loginFromToken := principalPayload.Login
		emailWeSentTo := svCDocInvite.AsString(Field_Email)
		if emailWeSentTo != loginFromToken && emailWeSentTo != principalPayload.Alias {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("invitation was sent to %s but current login is %s", emailWeSentTo, loginFromToken))
		}

		subjectID, subject, subjectExists, err := subjectByLogin(loginFromToken, args.State)
		if err != nil {
			return err
		}
		if subjectExists && subject.AsBool(appdef.SystemField_IsActive) {
			_, _, err = resolveControllingInvite(subjectID, subject, loginFromToken, principalPayload.Alias, inviteID, args.State)
			if err = controllingInviteResolutionError(err, loginFromToken); err != nil {
				return err
			}
		}

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return
		}
		svbCDocInvite.PutInt64(Field_InviteeProfileWSID, svPrincipal.AsInt64(sys.Storage_RequestSubject_Field_ProfileWSID))
		svbCDocInvite.PutInt32(authnz.Field_SubjectKind, svPrincipal.AsInt32(sys.Storage_RequestSubject_Field_Kind))
		svbCDocInvite.PutInt64(Field_Updated, tm.Now().UnixMilli())
		svbCDocInvite.PutChars(field_ActualLogin, loginFromToken)
		svbCDocInvite.PutInt32(Field_Version, 1)

		return
	}
}

func controllingInviteResolutionError(err error, canonicalLogin string) error {
	if errors.Is(err, ErrControllingInviteNotIdentified) {
		return coreutils.NewHTTPErrorf(http.StatusConflict, fmt.Sprintf(errControllingInviteNotIdentifiedMessageFormat, canonicalLogin))
	}
	return err
}
