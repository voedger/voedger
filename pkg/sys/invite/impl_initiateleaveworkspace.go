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
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
)

func provideCmdInitiateLeaveWorkspace(sr istructsmem.IStatelessResources, time timeu.ITime) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateLeaveWorkspace,
		execCmdInitiateLeaveWorkspace(time),
	))
}

func execCmdInitiateLeaveWorkspace(_ timeu.ITime) func(args istructs.ExecCommandArgs) (err error) {
	return func(args istructs.ExecCommandArgs) (err error) {
		skbPrincipal, err := args.State.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
		if err != nil {
			return
		}
		svPrincipal, err := args.State.MustExist(skbPrincipal)
		if err != nil {
			return
		}

		canonicalLogin := svPrincipal.AsString(sys.Storage_RequestSubject_Field_Name)
		subjectID, subject, subjectExists, err := subjectByLogin(canonicalLogin, args.State)
		if err != nil {
			return err
		}
		if !subjectExists || !subject.AsBool(appdef.SystemField_IsActive) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}

		principalPayload, err := payloads.GetPrincipalPayload(args.State.AppStructs().AppTokens(), svPrincipal.AsString(sys.Storage_RequestSubject_Field_Token))
		if err != nil {
			return err
		}
		inviteID, svCDocInvite, err := resolveControllingInvite(subjectID, subject, canonicalLogin, principalPayload.Alias, istructs.NullRecordID, args.State)
		if err = controllingInviteResolutionError(err, canonicalLogin); err != nil {
			return err
		}

		skbCDocInvite, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return err
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, inviteID)

		if !isValidInviteState(svCDocInvite.AsInt32(Field_State), qNameCmdInitiateLeaveWorkspace) {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrInviteStateInvalid)
		}

		// no-op CUD: keep UpdateValue so projector can discover InviteID from event.CUDs;
		// Version=1 marks this as a post-refactor event (ap.sys.ApplyInviteEvents skips Version==0)
		svbCDocInvite, err := args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)
		if err != nil {
			return err
		}
		svbCDocInvite.PutInt32(Field_Version, 1)

		return
	}
}
