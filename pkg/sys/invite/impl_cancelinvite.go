/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package invite

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func execCmdCancelInvite(cmdQName appdef.QName) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return err
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, args.ArgumentObject.AsRecordID(field_InviteID))
		svCDocInvite, ok, err := args.State.CanExist(skbCDocInvite)
		if err != nil {
			return err
		}
		if !ok {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteNotExists)
		}
		if !isValidInviteState(svCDocInvite.AsInt32(Field_State), cmdQName) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}
		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return err
		}
		svbCDocInvite.PutInt32(Field_Version, 1)
		return nil
	}
}
