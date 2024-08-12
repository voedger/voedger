/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideCmdInitiateJoinWorkspace(sr istructsmem.IStatelessResources, timeFunc coreutils.TimeFunc) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateJoinWorkspace,
		execCmdInitiateJoinWorkspace(timeFunc),
	))
}

func execCmdInitiateJoinWorkspace(timeFunc coreutils.TimeFunc) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, qNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, args.ArgumentObject.AsRecordID(field_InviteID))
		svCDocInvite, ok, err := args.State.CanExist(skbCDocInvite)
		if err != nil {
			return
		}
		if !ok {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteNotExists)
		}

		if !isValidInviteState(svCDocInvite.AsInt32(field_State), qNameCmdInitiateJoinWorkspace) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}
		if svCDocInvite.AsInt64(field_ExpireDatetime) < timeFunc().UnixMilli() {
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

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return
		}
		svbCDocInvite.PutInt64(field_InviteeProfileWSID, svPrincipal.AsInt64(sys.Storage_RequestSubject_Field_ProfileWSID))
		svbCDocInvite.PutInt32(authnz.Field_SubjectKind, svPrincipal.AsInt32(sys.Storage_RequestSubject_Field_Kind))
		svbCDocInvite.PutInt64(field_Updated, timeFunc().UnixMilli())
		svbCDocInvite.PutInt32(field_State, State_ToBeJoined)
		svbCDocInvite.PutChars(field_ActualLogin, svPrincipal.AsString(sys.Storage_RequestSubject_Field_Name))

		return
	}
}
