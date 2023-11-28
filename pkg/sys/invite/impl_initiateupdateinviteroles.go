/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"net/http"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideCmdInitiateUpdateInviteRoles(cfg *istructsmem.AppConfigType, timeFunc coreutils.TimeFunc) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdInitiateUpdateInviteRoles,
		execCmdInitiateUpdateInviteRoles(timeFunc),
	))
}

func execCmdInitiateUpdateInviteRoles(timeFunc coreutils.TimeFunc) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		if !coreutils.IsValidEmailTemplate(args.ArgumentObject.AsString(field_EmailTemplate)) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteTemplateInvalid)
		}

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

		if !isValidInviteState(svCDocInvite.AsInt32(field_State), qNameCmdInitiateUpdateInviteRoles) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteStateInvalid)
		}

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return
		}
		svbCDocInvite.PutInt32(field_State, State_ToUpdateRoles)
		svbCDocInvite.PutInt64(field_Updated, timeFunc().UnixMilli())

		return err
	}
}
