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
		if OnBeforeApplyJoinWorkspace != nil {
			OnBeforeApplyJoinWorkspace()
		}
		skbCDocInvite, err := s.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(sys.Storage_Record_Field_ID, event.ArgumentObject().AsRecordID(field_InviteID))
		svCDocInvite, err := s.MustExist(skbCDocInvite)
		if err != nil {
			return
		}
		if State(svCDocInvite.AsInt32(Field_State)) != State_ToBeJoined {
			return nil
		}
		if OnAfterGuardApplyJoinWorkspace != nil {
			OnAfterGuardApplyJoinWorkspace()
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

		// No early skip here - all operations below are idempotent, ensuring completion on projector retries:
		// - CreateJoinedWorkspace: checks if exists, updates if so (see impl_createjoinedworkspace.go)
		// - Subject create/reactivate: handled by existingSubjectID check
		// - Invite update: idempotent state transition

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
		switch {
		case existingSubjectID == istructs.NullRecordID:
			// Create new Subject
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
				svCDocInvite.AsString(field_ActualLogin), svCDocInvite.AsString(Field_Roles), svCDocInvite.AsInt32(authnz.Field_SubjectKind),
				svCDocInvite.AsInt64(Field_InviteeProfileWSID))
		case !isActive:
			// Reactivate inactive Subject - first activate, then update Roles (can't update both in one call)
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, existingSubjectID)
			_, err = federation.Func(
				fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
				body,
				httpu.WithAuthorizeBy(token),
				httpu.WithDiscardResponse())
			if err != nil {
				return
			}
			fallthrough
		default:
			// Subject already active (retry scenario) - just update Roles
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, existingSubjectID, svCDocInvite.AsString(Field_Roles))
		}
		resp, err := federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		// Update cdoc.sys.Invite State=Joined via validated command
		subjectID := resp.NewID()
		if existingSubjectID != istructs.NullRecordID {
			subjectID = 0
		}
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CompleteJoinWorkspace", appQName, event.Workspace()),
			fmt.Sprintf(`{"args":{"InviteID":%d,"SubjectID":%d,"Updated":%d}}`,
				svCDocInvite.AsRecordID(appdef.SystemField_ID), subjectID, time.Now().UnixMilli()),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())

		return err
	}
}
