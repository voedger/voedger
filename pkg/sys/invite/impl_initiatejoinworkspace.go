/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideCmdInitiateJoinWorkspace(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, timeFunc func() time.Time) {
	pars := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "InitiateJoinWorkspaceParams"))
	pars.AddField(field_InviteID, appdef.DataKind_RecordID, true).
		AddField(field_VerificationCode, appdef.DataKind_string, true)
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdInitiateJoinWorkspace, pars.QName(), appdef.NullQName, appdef.NullQName,
		execCmdInitiateJoinWorkspace(timeFunc),
	))
}

func execCmdInitiateJoinWorkspace(timeFunc func() time.Time) func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	return func(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
		skbCDocInvite, err := args.State.KeyBuilder(state.RecordsStorage, qNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(state.Field_ID, args.ArgumentObject.AsRecordID(field_InviteID))
		svCDocInvite, ok, err := args.State.CanExist(skbCDocInvite)
		if err != nil {
			return
		}
		if !ok {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteNotExists)
		}

		if !isValidInviteState(svCDocInvite.AsInt32(field_State), qNameCmdInitiateJoinWorkspace) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteStateInvalid)
		}
		if svCDocInvite.AsInt64(field_ExpireDatetime) < timeFunc().UnixMilli() {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteExpired)
		}
		if svCDocInvite.AsString(field_VerificationCode) != args.ArgumentObject.AsString(field_VerificationCode) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteVerificationCodeInvalid)
		}

		skbPrincipal, err := args.State.KeyBuilder(state.SubjectStorage, appdef.NullQName)
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
		svbCDocInvite.PutInt64(field_InviteeProfileWSID, svPrincipal.AsInt64(state.Field_ProfileWSID))
		svbCDocInvite.PutInt32(sysshared.Field_SubjectKind, svPrincipal.AsInt32(state.Field_Kind))
		svbCDocInvite.PutInt64(field_Updated, timeFunc().UnixMilli())
		svbCDocInvite.PutInt32(field_State, State_ToBeJoined)

		return
	}
}
