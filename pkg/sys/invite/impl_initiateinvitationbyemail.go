/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideCmdInitiateInvitationByEMail(sr istructsmem.IStatelessResources, time timeu.ITime) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateInvitationByEMail,
		execCmdInitiateInvitationByEMail(time),
	))
}

// called in the workspace that we're inviting to.
// [~server.invites.invite/c.sys.Workspace.InitiateInvitationByEMail~impl]
func execCmdInitiateInvitationByEMail(tm timeu.ITime) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		if !coreutils.IsValidEmailTemplate(args.ArgumentObject.AsString(field_EmailTemplate)) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteTemplateInvalid)
		}

		cmdInitiateInvitation_ArgEmail := args.ArgumentObject.AsString(field_Email)
		// do not check if the login from token exists in subjects, see https://github.com/voedger/voedger/issues/3698
		// because login is Inviter here, not Invitee
		existingSubjectID, err := SubjectExistsByLogin(cmdInitiateInvitation_ArgEmail, args.State)
		if err != nil {
			return
		}

		if err := coreutils.ValidateEMail(cmdInitiateInvitation_ArgEmail); err != nil {
			return err
		}

		skbViewInviteIndex, err := args.State.KeyBuilder(sys.Storage_View, qNameViewInviteIndex)
		if err != nil {
			return
		}
		skbViewInviteIndex.PutInt32(field_Dummy, value_Dummy_One)
		skbViewInviteIndex.PutString(Field_Login, args.ArgumentObject.AsString(field_Email))
		svViewInviteIndex, ok, err := args.State.CanExist(skbViewInviteIndex)
		if err != nil {
			return
		}

		if ok {
			skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
			if err != nil {
				return err
			}
			skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, svViewInviteIndex.AsRecordID(field_InviteID))
			svCDocInvite, err := args.State.MustExist(skbCDocInvite)
			if err != nil {
				return err
			}

			inviteState := State(svCDocInvite.AsInt32(Field_State))
			if existingSubjectID > 0 && !reInviteAllowedForState[inviteState] {
				return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf(`%w %s`, ErrReInviteNotAllowedForState, inviteState))
			}

			if !isValidInviteState(svCDocInvite.AsInt32(Field_State), qNameCmdInitiateInvitationByEMail) {
				return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
			}

			svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
			if err != nil {
				return err
			}
			svbCDocInvite.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
			svbCDocInvite.PutInt64(field_ExpireDatetime, args.ArgumentObject.AsInt64(field_ExpireDatetime))
			svbCDocInvite.PutInt32(Field_State, int32(State_ToBeInvited))
			svbCDocInvite.PutInt64(Field_Updated, tm.Now().UnixMilli())
			svbCDocInvite.PutString(field_ActualLogin, "") // to be filled with Invitee's login by ap.sys.Apply

			return nil
		}

		skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return err
		}
		svbCDocInvite, err := args.Intents.NewValue(skbCDocInvite)
		if err != nil {
			return err
		}
		now := tm.Now().UnixMilli()
		svbCDocInvite.PutRecordID(appdef.SystemField_ID, istructs.RecordID(1))
		svbCDocInvite.PutString(Field_Login, args.ArgumentObject.AsString(field_Email))
		svbCDocInvite.PutString(field_Email, args.ArgumentObject.AsString(field_Email))
		svbCDocInvite.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
		svbCDocInvite.PutInt64(field_ExpireDatetime, args.ArgumentObject.AsInt64(field_ExpireDatetime))
		svbCDocInvite.PutInt64(field_Created, now)
		svbCDocInvite.PutInt64(Field_Updated, now)
		svbCDocInvite.PutInt32(Field_State, int32(State_ToBeInvited))
		// do not fill cdoc.sys.Invite.ActualLogin because it must be Invitee's login. It is unknown here

		return
	}
}
