/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"
	"net/url"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/collection"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func ProvideAsyncProjectorApplyJoinWorkspaceFactory(timeFunc func() time.Time, federationURL func() *url.URL, appQName istructs.AppQName, tokens itokens.ITokens) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameAPApplyJoinWorkspace,
			EventsFilter: []appdef.QName{qNameCmdInitiateJoinWorkspace},
			Func:         applyJoinWorkspace(timeFunc, federationURL, appQName, tokens),
		}
	}
}

func applyJoinWorkspace(timeFunc func() time.Time, federationURL func() *url.URL, appQName istructs.AppQName, tokens itokens.ITokens) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		skbCDocInvite, err := s.KeyBuilder(state.RecordsStorage, qNameCDocInvite)
		if err != nil {
			return
		}
		skbCDocInvite.PutRecordID(state.Field_ID, event.ArgumentObject().AsRecordID(field_InviteID))
		svCDocInvite, err := s.MustExist(skbCDocInvite)
		if err != nil {
			return
		}

		skbCDocWorkspaceDescriptor, err := s.KeyBuilder(state.RecordsStorage, sysshared.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return err
		}
		skbCDocWorkspaceDescriptor.PutQName(state.Field_Singleton, sysshared.QNameCDocWorkspaceDescriptor)
		svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
		if err != nil {
			return
		}

		token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}
		_, err = coreutils.FederationFunc(
			federationURL(),
			fmt.Sprintf("api/%s/%d/c.sys.CreateJoinedWorkspace", appQName, svCDocInvite.AsInt64(field_InviteeProfileWSID)),
			fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d,"WSName":"%s"}}`,
				svCDocInvite.AsString(Field_Roles), event.Workspace(), svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName)),
			coreutils.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		//Find cdoc.sys.Subject by cdoc.air.Invite
		skbViewCollection, err := s.KeyBuilder(state.ViewRecordsStorage, collection.QNameViewCollection)
		if err != nil {
			return
		}
		skbViewCollection.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
		skbViewCollection.PutQName(collection.Field_DocQName, sysshared.QNameCDocSubject)

		var svCDocSubject istructs.IStateValue
		err = s.Read(skbViewCollection, func(key istructs.IKey, value istructs.IStateValue) (err error) {
			if svCDocSubject != nil || svCDocInvite.AsRecordID(field_SubjectID) == istructs.NullRecordID {
				return
			}
			if svCDocInvite.AsRecordID(field_SubjectID) == value.AsRecordID(appdef.SystemField_ID) {
				svCDocSubject = value
			}
			return err
		})
		if err != nil {
			return
		}

		var body string
		//Store cdoc.sys.Subject
		if svCDocSubject == nil {
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
				svCDocInvite.AsString(Field_Login), svCDocInvite.AsString(Field_Roles), svCDocInvite.AsInt32(sysshared.Field_SubjectKind),
				svCDocInvite.AsInt64(field_InviteeProfileWSID))
		} else {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, svCDocSubject.AsRecordID(appdef.SystemField_ID), svCDocInvite.AsString(Field_Roles))
		}
		resp, err := coreutils.FederationFunc(
			federationURL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			coreutils.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		//Store cdoc.sys.Invite
		//TODO why Login update???
		if svCDocSubject == nil {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"SubjectID":%d,"Updated":%d}}]}`, svCDocInvite.AsRecordID(appdef.SystemField_ID), State_Joined, resp.NewID(), timeFunc().UnixMilli())
		} else {
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d}}]}`, svCDocInvite.AsRecordID(appdef.SystemField_ID), State_Joined, timeFunc().UnixMilli())
		}
		_, err = coreutils.FederationFunc(
			federationURL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			body,
			coreutils.WithAuthorizeBy(token))

		return err
	}
}
