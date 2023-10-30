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

func provideCmdCancelSentInvite(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, timeFunc coreutils.TimeFunc) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdCancelSentInvite,
		appdef.NullQName,
		appdef.NullQName,
		appdef.NullQName,
		execCmdCancelSentInvite(timeFunc),
	))
}

func execCmdCancelSentInvite(timeFunc coreutils.TimeFunc) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		skbCDocInvite, err := args.State.KeyBuilder(state.Record, qNameCDocInvite)
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

		if !isValidInviteState(svCDocInvite.AsInt32(field_State), qNameCmdCancelSentInvite) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteStateInvalid)
		}

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return
		}
		svbCDocInvite.PutInt64(field_Updated, timeFunc().UnixMilli())
		svbCDocInvite.PutInt32(field_State, State_Cancelled)

		return
	}
}
