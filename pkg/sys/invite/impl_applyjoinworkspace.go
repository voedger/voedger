/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/collection"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideAsyncProjectorApplyJoinWorkspaceFactory(timeFunc coreutils.TimeFunc, federation coreutils.IFederation, appQName istructs.AppQName, tokens itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPApplyJoinWorkspace,
			Func: applyJoinWorkspace(timeFunc, federation, appQName, tokens),
		}
	}
}

func applyJoinWorkspace(timeFunc coreutils.TimeFunc, federation coreutils.IFederation, appQName istructs.AppQName, tokens itokens.ITokens) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		// it is AFTER EXECUTE ON (InitiateJoinWorkspace) so no doc checking here
		skbCDocInvite, err := s.KeyBuilder(state.Record, qNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(state.Field_ID, event.ArgumentObject().AsRecordID(field_InviteID))
		svCDocInvite, err := s.MustExist(skbCDocInvite)
		if err != nil {
			return
		}

		login := svCDocInvite.AsString(Field_Login)
		subjectExists, err := SubjectExistByLogin(login, s) // for backward compatibility
		if err == nil && !subjectExists {

			actualLogin := svCDocInvite.AsString(field_ActualLogin)
			subjectExists, err = SubjectExistByLogin(actualLogin, s)
		}
		if err != nil {
			// notest
			return err
		}
		if subjectExists && svCDocInvite.AsInt32(field_State) == State_Joined {
			// cdoc.sys.Subject eists by login and invite state is any of [State_ToBeInvited, State_Invited, State_ToBeJoined, State_Joined, State_ToUpdateRoles] -> do nothing
			// otherwise - consider the workspace is joining again
			// see https://github.com/voedger/voedger/issues/1107
			return nil
		}

		skbCDocWorkspaceDescriptor, err := s.KeyBuilder(state.Record, authnz.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return err
		}
		skbCDocWorkspaceDescriptor.PutQName(state.Field_Singleton, authnz.QNameCDocWorkspaceDescriptor)
		svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
		if err != nil {
			return
		}

		token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}
		_, err = coreutils.FederationFunc(
			federation.URL(),
			fmt.Sprintf("api/%s/%d/c.sys.CreateJoinedWorkspace", appQName, svCDocInvite.AsInt64(field_InviteeProfileWSID)),
			fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d,"WSName":"%s"}}`,
				svCDocInvite.AsString(Field_Roles), event.Workspace(), svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName)),
			coreutils.WithAuthorizeBy(token),
			coreutils.WithDiscardResponse(),
		)
		if err != nil {
			return
		}

		//Find cdoc.sys.Subject by cdoc.air.Invite
		skbViewCollection, err := s.KeyBuilder(state.View, collection.QNameCollectionView)
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
		//Store cdoc.sys.Subject
		if svCDocSubject == nil {
			// svCDocInvite.AsString(Field_Login) is actually c.sys.InitiateInvitationByEMail.Email
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
				svCDocInvite.AsString(field_ActualLogin), svCDocInvite.AsString(Field_Roles), svCDocInvite.AsInt32(authnz.Field_SubjectKind),
				svCDocInvite.AsInt64(field_InviteeProfileWSID))
		} else {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`,
				svCDocSubject.AsRecordID(appdef.SystemField_ID), svCDocInvite.AsString(Field_Roles))
		}
		resp, err := coreutils.FederationFunc(
			federation.URL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			coreutils.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		//Store cdoc.sys.Invite
		//TODO why Login update???
		if svCDocSubject == nil {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"SubjectID":%d,"Updated":%d}}]}`,
				svCDocInvite.AsRecordID(appdef.SystemField_ID), State_Joined, resp.NewID(), timeFunc().UnixMilli())
		} else {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d}}]}`,
				svCDocInvite.AsRecordID(appdef.SystemField_ID), State_Joined, timeFunc().UnixMilli())
		}
		_, err = coreutils.FederationFunc(
			federation.URL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			coreutils.WithAuthorizeBy(token))

		return err
	}
}
