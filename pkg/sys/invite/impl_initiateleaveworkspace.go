/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideCmdInitiateLeaveWorkspace(sr istructsmem.IStatelessResources, time timeu.ITime) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateLeaveWorkspace,
		execCmdInitiateLeaveWorkspace(time),
	))
}

func execCmdInitiateLeaveWorkspace(time timeu.ITime) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		skbPrincipal, err := args.State.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
		if err != nil {
			return
		}
		svPrincipal, err := args.State.MustExist(skbPrincipal)
		if err != nil {
			return
		}

		skbViewInviteIndex, err := args.State.KeyBuilder(sys.Storage_View, qNameViewInviteIndex)
		if err != nil {
			return
		}
		skbViewInviteIndex.PutInt32(field_Dummy, value_Dummy_One)
		skbViewInviteIndex.PutString(Field_Login, svPrincipal.AsString(sys.Storage_RequestSubject_Field_Name))
		svViewInviteIndex, err := args.State.MustExist(skbViewInviteIndex)
		if err != nil {
			return
		}

		skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return err
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, svViewInviteIndex.AsRecordID(field_InviteID))
		svCDocInvite, err := args.State.MustExist(skbCDocInvite)
		if err != nil {
			return err
		}

		if !isValidInviteState(svCDocInvite.AsInt32(Field_State), qNameCmdInitiateLeaveWorkspace) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return err
		}
		svbCDocInvite.PutInt32(Field_State, int32(State_ToBeLeft))
		svbCDocInvite.PutInt64(Field_Updated, time.Now().UnixMilli())
		svbCDocInvite.PutBool(appdef.SystemField_IsActive, false)

		return
	}
}
