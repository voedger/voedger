/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func asyncProjectorApplyJoinWorkspace(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyJoinWorkspace,
		Func: applyJoinWorkspace(time, federation, tokens),
	}
}

func applyJoinWorkspace(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		// it is AFTER EXECUTE ON (InitiateJoinWorkspace) so no doc checking here
		skbCDocInvite, err := s.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, event.ArgumentObject().AsRecordID(field_InviteID))
		svCDocInvite, err := s.MustExist(skbCDocInvite)
		if err != nil {
			return
		}

		// Check if Subject exists for this user
		// ActualLogin = user's login from auth token (set by InitiateJoinWorkspace)
		// Note: Currently ActualLogin always equals Login because InitiateJoinWorkspace validates they match (earlier it wasn't required)
		login := svCDocInvite.AsString(field_ActualLogin)
		existingSubjectID, isActive, err := SubjectExistsByLogin(login, s)
		if err != nil {
			// notest
			return err
		}

		if isActive {
			// Active Subject already exists -> skip
			// see https://github.com/voedger/voedger/issues/1107
			logger.Info(fmt.Sprintf(`skip aproj.sys.ApplyJoinWorkspace: active Subject %d already exists for login "%s"`, existingSubjectID, login))
			return nil
		}

		skbCDocWorkspaceDescriptor, err := s.KeyBuilder(sys.Storage_Record, appdef.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return err
		}
		skbCDocWorkspaceDescriptor.PutQName(sys.Storage_Record_Field_Singleton, appdef.QNameCDocWorkspaceDescriptor)
		svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
		if err != nil {
			return
		}

		appQName := s.App()

		token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CreateJoinedWorkspace", appQName, svCDocInvite.AsInt64(Field_InviteeProfileWSID)),
			fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d,"WSName":%q}}`,
				svCDocInvite.AsString(Field_Roles), event.Workspace(), svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName)),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse(),
		)
		if err != nil {
			return
		}

		var body string
		// Store cdoc.sys.Subject
		if existingSubjectID == istructs.NullRecordID {
			// Create new Subject
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
				svCDocInvite.AsString(field_ActualLogin), svCDocInvite.AsString(Field_Roles), svCDocInvite.AsInt32(authnz.Field_SubjectKind),
				svCDocInvite.AsInt64(Field_InviteeProfileWSID))
		} else {
			// Reactivate existing Subject - first activate, then update Roles (can't update both in one call)
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, existingSubjectID)
			_, err = federation.Func(
				fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
				body,
				httpu.WithAuthorizeBy(token),
				httpu.WithDiscardResponse())
			if err != nil {
				return
			}
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, existingSubjectID, svCDocInvite.AsString(Field_Roles))
		}
		resp, err := federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		// Store cdoc.sys.Invite
		if existingSubjectID == istructs.NullRecordID {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"SubjectID":%d,"Updated":%d}}]}`,
				svCDocInvite.AsRecordID(appdef.SystemField_ID), State_Joined, resp.NewID(), time.Now().UnixMilli())
		} else {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d}}]}`,
				svCDocInvite.AsRecordID(appdef.SystemField_ID), State_Joined, time.Now().UnixMilli())
		}
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())

		return err
	}
}
