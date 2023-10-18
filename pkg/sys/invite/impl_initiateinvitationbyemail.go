/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideCmdInitiateInvitationByEMail(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, timeFunc coreutils.TimeFunc) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdInitiateInvitationByEMail,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "InitiateInvitationByEMailParams")).
			AddField(field_Email, appdef.DataKind_string, true).
			AddField(Field_Roles, appdef.DataKind_string, true).
			AddField(field_ExpireDatetime, appdef.DataKind_int64, true).
			AddField(field_EmailTemplate, appdef.DataKind_string, true).
			AddField(field_EmailSubject, appdef.DataKind_string, true).(appdef.IType).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdInitiateInvitationByEMail(timeFunc),
	))
}

func execCmdInitiateInvitationByEMail(timeFunc coreutils.TimeFunc) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		if !coreutils.IsValidEmailTemplate(args.ArgumentObject.AsString(field_EmailTemplate)) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteTemplateInvalid)
		}

		skbViewInviteIndex, err := args.State.KeyBuilder(state.View, qNameViewInviteIndex)
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
			skbCDocInvite, err := args.State.KeyBuilder(state.Record, qNameCDocInvite)
			if err != nil {
				return err
			}
			skbCDocInvite.PutRecordID(state.Field_ID, svViewInviteIndex.AsRecordID(field_InviteID))
			svCDocInvite, err := args.State.MustExist(skbCDocInvite)
			if err != nil {
				return err
			}

			if !isValidInviteState(svCDocInvite.AsInt32(field_State), qNameCmdInitiateInvitationByEMail) {
				return coreutils.NewHTTPError(http.StatusBadRequest, errInviteStateInvalid)
			}

			svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
			if err != nil {
				return err
			}
			svbCDocInvite.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
			svbCDocInvite.PutInt64(field_ExpireDatetime, args.ArgumentObject.AsInt64(field_ExpireDatetime))
			svbCDocInvite.PutInt32(field_State, State_ToBeInvited)
			svbCDocInvite.PutInt64(field_Updated, timeFunc().UnixMilli())

			return nil
		}

		skbCDocInvite, err := args.State.KeyBuilder(state.Record, qNameCDocInvite)
		if err != nil {
			return err
		}
		svbCDocInvite, err := args.Intents.NewValue(skbCDocInvite)
		if err != nil {
			return err
		}
		now := timeFunc().UnixMilli()
		svbCDocInvite.PutRecordID(appdef.SystemField_ID, istructs.RecordID(1))
		svbCDocInvite.PutString(Field_Login, args.ArgumentObject.AsString(field_Email))
		svbCDocInvite.PutString(field_Email, args.ArgumentObject.AsString(field_Email))
		svbCDocInvite.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
		svbCDocInvite.PutInt64(field_ExpireDatetime, args.ArgumentObject.AsInt64(field_ExpireDatetime))
		svbCDocInvite.PutInt64(field_Created, now)
		svbCDocInvite.PutInt64(field_Updated, now)
		svbCDocInvite.PutInt32(field_State, State_ToBeInvited)

		return
	}
}
