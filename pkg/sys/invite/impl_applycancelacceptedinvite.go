/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
)

func asyncProjectorApplyCancelAcceptedInvite(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyCancelAcceptedInvite,
		Func: applyCancelAcceptedInvite(time, federation, tokens),
	}
}

// AFTER EXEC c.sys.InitiateCancelAcceptedInvite
func applyCancelAcceptedInvite(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		skbCDocInvite, err := s.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, event.ArgumentObject().AsRecordID(field_InviteID))
		svCDocInvite, err := s.MustExist(skbCDocInvite)
		if err != nil {
			return
		}

		skbCDocSubject, err := s.KeyBuilder(sys.Storage_Record, QNameCDocSubject)
		if err != nil {
			return
		}
		skbCDocSubject.PutRecordID(sys.Storage_Record_Field_ID, svCDocInvite.AsRecordID(field_SubjectID))
		svCDocSubject, err := s.MustExist(skbCDocSubject)
		if err != nil {
			return
		}

		appQName := s.App()

		token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}

		// Update subject
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, svCDocSubject.AsRecordID(appdef.SystemField_ID)),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())
		if err != nil {
			return
		}

		// Deactivate joined workspace
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.DeactivateJoinedWorkspace", appQName, svCDocInvite.AsInt64(Field_InviteeProfileWSID)),
			fmt.Sprintf(`{"args":{"InvitingWorkspaceWSID":%d}}`, event.Workspace()),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())
		if err != nil {
			return
		}

		// Update invite
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d}}]}`, event.ArgumentObject().AsRecordID(field_InviteID), State_Cancelled, time.Now().UnixMilli()),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())

		return err
	}
}
