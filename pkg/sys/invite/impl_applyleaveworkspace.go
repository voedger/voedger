/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
)

func asyncProjectorApplyLeaveWorkspace(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyLeaveWorkspace,
		Func: applyLeaveWorkspace(time, federation, tokens),
	}
}

func applyLeaveWorkspace(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) error {
		for rec := range event.CUDs {
			//TODO additional check that CUD only once?
			if rec.QName() != qNameCDocInvite {
				continue
			}

			skbCDocInvite, err := s.KeyBuilder(sys.Storage_Record, qNameCDocInvite)
			if err != nil {
				return err
			}
			skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, rec.ID())
			svCDocInvite, err := s.MustExist(skbCDocInvite)
			if err != nil {
				return err
			}

			skbCDocSubject, err := s.KeyBuilder(sys.Storage_Record, QNameCDocSubject)
			if err != nil {
				return err
			}
			skbCDocSubject.PutRecordID(sys.Storage_Record_Field_ID, svCDocInvite.AsRecordID(field_SubjectID))
			svCDocSubject, err := s.MustExist(skbCDocSubject)
			if err != nil {
				return err
			}

			appQName := s.App()

			token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
			if err != nil {
				return err
			}

			//Update subject
			if _, err = federation.Func(
				fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
				fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, svCDocSubject.AsRecordID(appdef.SystemField_ID)),
				coreutils.WithAuthorizeBy(token),
				coreutils.WithDiscardResponse()); err != nil {
				return err
			}

			//Deactivate joined workspace
			if _, err = federation.Func(
				fmt.Sprintf("api/%s/%d/c.sys.DeactivateJoinedWorkspace", appQName, svCDocInvite.AsInt64(field_InviteeProfileWSID)),
				fmt.Sprintf(`{"args":{"InvitingWorkspaceWSID":%d}}`, event.Workspace()),
				coreutils.WithAuthorizeBy(token),
				coreutils.WithDiscardResponse()); err != nil {
				return err
			}

			//Update invite
			if _, err = federation.Func(
				fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
				fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d}}]}`, rec.ID(), State_Left, time.Now().UnixMilli()),
				coreutils.WithAuthorizeBy(token),
				coreutils.WithDiscardResponse()); err != nil {
				return err
			}
		}
		return nil
	}
}
