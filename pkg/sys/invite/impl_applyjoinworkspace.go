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
	"github.com/voedger/voedger/pkg/sys/collection"
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

		login := svCDocInvite.AsString(Field_Login)
		subjectExistsByActualLogin := false
		existingSubjectID, err := SubjectExistsByLogin(login, s) // for backward compatibility
		subjectExistsByLogin := existingSubjectID > 0
		if err == nil && !subjectExistsByLogin {
			login = svCDocInvite.AsString(field_ActualLogin)
			existingSubjectID, err = SubjectExistsByLogin(login, s)
			subjectExistsByActualLogin = existingSubjectID > 0
		}
		if err != nil {
			// notest
			return err
		}

		if subjectExistsByLogin || subjectExistsByActualLogin {
			// cdoc.sys.Subject exists by login -> skip
			// see https://github.com/voedger/voedger/issues/1107
			// && svCDocInvite.AsInt32(Field_State) == State_Joined -> insert cdoc.sys.Subject with an existing login -> unique violation -> the projector stuck
			fieldName := "cdoc.sys.Invite.Login"
			if subjectExistsByActualLogin {
				fieldName = "cdoc.sys.Invite.ActualLogin"
			}
			logger.Info(fmt.Sprintf(`skip aproj.sys.ApplyJoinWorkspace because cdoc.sys.SubjectID.%d exists already by %s "%s"`, existingSubjectID, fieldName, login))
			return nil
		}

		skbCDocWorkspaceDescriptor, err := s.KeyBuilder(sys.Storage_Record, authnz.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return err
		}
		skbCDocWorkspaceDescriptor.PutQName(sys.Storage_Record_Field_Singleton, authnz.QNameCDocWorkspaceDescriptor)
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

		// Find cdoc.sys.Subject by cdoc.air.Invite
		skbViewCollection, err := s.KeyBuilder(sys.Storage_View, collection.QNameCollectionView)
		if err != nil {
			return
		}
		skbViewCollection.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
		skbViewCollection.PutQName(collection.Field_DocQName, QNameCDocSubject)

		var svCDocSubject istructs.IStateValue
		if svCDocInvite.AsRecordID(field_SubjectID) != istructs.NullRecordID {
			err = s.Read(skbViewCollection, func(key istructs.IKey, value istructs.IStateValue) (err error) {
				if svCDocSubject != nil {
					return nil
				}
				if svCDocInvite.AsRecordID(field_SubjectID) == value.AsRecordID(appdef.SystemField_ID) {
					svCDocSubject = value
				}
				return nil
			})
			if err != nil {
				return
			}
		}

		var body string
		// Store cdoc.sys.Subject
		if svCDocSubject == nil {
			// svCDocInvite.AsString(Field_Login) is actually c.sys.InitiateInvitationByEMail.Email
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
				svCDocInvite.AsString(field_ActualLogin), svCDocInvite.AsString(Field_Roles), svCDocInvite.AsInt32(authnz.Field_SubjectKind),
				svCDocInvite.AsInt64(Field_InviteeProfileWSID))
		} else {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`,
				svCDocSubject.AsRecordID(appdef.SystemField_ID), svCDocInvite.AsString(Field_Roles))
		}
		resp, err := federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		//Store cdoc.sys.Invite
		//TODO why Login update???
		if svCDocSubject == nil {
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
