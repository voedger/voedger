/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func ProvideAsyncProjectorApplyUpdateInviteRolesFactory(timeFunc func() time.Time, federation coreutils.IFederation, appQName istructs.AppQName, tokens itokens.ITokens, smtpCfg smtp.Cfg) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameAPApplyUpdateInviteRoles,
			EventsFilter: []appdef.QName{qNameCmdInitiateUpdateInviteRoles},
			Func:         applyUpdateInviteRolesProjector(timeFunc, federation, appQName, tokens, smtpCfg),
			NonBuffered:  true,
		}
	}
}

func applyUpdateInviteRolesProjector(timeFunc func() time.Time, federation coreutils.IFederation, appQName istructs.AppQName, tokens itokens.ITokens, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
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

		skbCDocSubject, err := s.KeyBuilder(state.RecordsStorage, QNameCDocSubject)
		if err != nil {
			return
		}
		skbCDocSubject.PutRecordID(state.Field_ID, svCDocInvite.AsRecordID(field_SubjectID))
		svCDocSubject, err := s.MustExist(skbCDocSubject)
		if err != nil {
			return
		}

		token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}

		//Update subject
		_, err = coreutils.FederationFunc(
			federationURL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, svCDocSubject.AsRecordID(appdef.SystemField_ID), event.ArgumentObject().AsString(Field_Roles)),
			coreutils.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		//Update joined workspace roles
		_, err = coreutils.FederationFunc(
			federationURL(),
			fmt.Sprintf("api/%s/%d/c.sys.UpdateJoinedWorkspaceRoles", appQName, svCDocInvite.AsInt64(field_InviteeProfileWSID)),
			fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d}}`, event.ArgumentObject().AsString(Field_Roles), event.Workspace()),
			coreutils.WithAuthorizeBy(token))
		if err != nil {
			return
		}

		emailTemplate := coreutils.TruncateEmailTemplate(event.ArgumentObject().AsString(field_EmailTemplate))

		replacer := strings.NewReplacer(EmailTemplatePlaceholder_Roles, event.ArgumentObject().AsString(Field_Roles))

		//Send roles update email
		skbSendMail, err := s.KeyBuilder(state.SendMailStorage, appdef.NullQName)
		if err != nil {
			return
		}
		skbSendMail.PutString(state.Field_Subject, event.ArgumentObject().AsString(field_EmailSubject))
		skbSendMail.PutString(state.Field_To, svCDocInvite.AsString(field_Email))
		skbSendMail.PutString(state.Field_Body, replacer.Replace(emailTemplate))
		skbSendMail.PutString(state.Field_From, EmailFrom)
		skbSendMail.PutString(state.Field_Host, smtpCfg.Host)
		skbSendMail.PutInt32(state.Field_Port, smtpCfg.Port)
		skbSendMail.PutString(state.Field_Username, smtpCfg.Username)

		pwd := ""
		if !coreutils.IsTest() {
			skbAppSecretsStorage, err := s.KeyBuilder(state.AppSecretsStorage, appdef.NullQName)
			if err != nil {
				return err
			}
			skbAppSecretsStorage.PutString(state.Field_Secret, smtpCfg.PwdSecret)
			svAppSecretsStorage, err := s.MustExist(skbAppSecretsStorage)
			if err != nil {
				return err
			}
			pwd = svAppSecretsStorage.AsString("")
		}
		skbSendMail.PutString(state.Field_Password, pwd)

		_, err = intents.NewValue(skbSendMail)
		if err != nil {
			return
		}

		//Update invite
		_, err = coreutils.FederationFunc(
			federationURL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d,"Roles":"%s"}}]}`, event.ArgumentObject().AsRecordID(field_InviteID), State_Joined, timeFunc().UnixMilli(), event.ArgumentObject().AsString(Field_Roles)),
			coreutils.WithAuthorizeBy(token))

		return err
	}
}
