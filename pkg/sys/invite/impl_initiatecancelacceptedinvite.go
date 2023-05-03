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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideCmdInitiateCancelAcceptedInvite(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, timeFunc func() time.Time) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdInitiateCancelAcceptedInvite,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "InitiateCancelAcceptedInviteParams"), appdef.DefKind_Object).
			AddField(field_InviteID, appdef.DataKind_RecordID, true).
			QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdInitiateCancelAcceptedInvite(timeFunc),
	))
}

func execCmdInitiateCancelAcceptedInvite(timeFunc func() time.Time) func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	return func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
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

		if !isValidInviteState(svCDocInvite.AsInt32(field_State), qNameCmdInitiateCancelAcceptedInvite) {
			return coreutils.NewHTTPError(http.StatusBadRequest, errInviteStateInvalid)
		}

		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return
		}
		svbCDocInvite.PutInt64(field_Updated, timeFunc().UnixMilli())
		svbCDocInvite.PutInt32(field_State, State_ToBeCancelled)

		return err
	}
}
