/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideCmdCompleteJoinWorkspace(sr istructsmem.IStatelessResources) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdCompleteJoinWorkspace,
		execCmdCompleteJoinWorkspace,
	))
}

func execCmdCompleteJoinWorkspace(args istructs.ExecCommandArgs) (err error) {
	skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
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
	if State(svCDocInvite.AsInt32(Field_State)) != State_ToBeJoined {
		return coreutils.NewHTTPError(http.StatusConflict, ErrInviteStateInvalid)
	}

	svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
	if err != nil {
		return
	}
	svbCDocInvite.PutInt32(Field_State, int32(State_Joined))
	svbCDocInvite.PutInt64(Field_Updated, args.ArgumentObject.AsInt64(Field_Updated))
	subjectID := args.ArgumentObject.AsRecordID(field_SubjectID)
	if subjectID != istructs.NullRecordID {
		svbCDocInvite.PutRecordID(field_SubjectID, subjectID)
	}

	return
}
