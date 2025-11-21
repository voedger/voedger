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

func provideCmdInitiateCancelAcceptedInvite(sr istructsmem.IStatelessResources, time timeu.ITime) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateCancelAcceptedInvite,
		execCmdInitiateCancelAcceptedInvite(time),
	))
}

func execCmdInitiateCancelAcceptedInvite(time timeu.ITime) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
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

		if !isValidInviteState(svCDocInvite.AsInt32(Field_State), qNameCmdInitiateCancelAcceptedInvite) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return
		}
		svbCDocInvite.PutInt64(Field_Updated, time.Now().UnixMilli())
		svbCDocInvite.PutInt32(Field_State, int32(State_ToBeCancelled))

		return err
	}
}
