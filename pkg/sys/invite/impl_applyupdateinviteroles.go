/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func asyncProjectorApplyUpdateInviteRoles(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyUpdateInviteRoles,
		Func: applyUpdateInviteRolesProjector(time, federation, tokens, smtpCfg),
	}
}

func applyUpdateInviteRolesProjector(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
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

		//Update subject
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, svCDocSubject.AsRecordID(appdef.SystemField_ID), event.ArgumentObject().AsString(Field_Roles)),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())
		if err != nil {
			return
		}

		//Update joined workspace roles
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.UpdateJoinedWorkspaceRoles", appQName, svCDocInvite.AsInt64(Field_InviteeProfileWSID)),
			fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d}}`, event.ArgumentObject().AsString(Field_Roles), event.Workspace()),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())
		if err != nil {
			return
		}

		emailTemplate := coreutils.TruncateEmailTemplate(event.ArgumentObject().AsString(field_EmailTemplate))

		replacer := strings.NewReplacer(EmailTemplatePlaceholder_Roles, event.ArgumentObject().AsString(Field_Roles))
		//Send roles update email
		skbSendMail, err := s.KeyBuilder(sys.Storage_SendMail, appdef.NullQName)
		if err != nil {
			return
		}
		skbSendMail.PutString(sys.Storage_SendMail_Field_Subject, event.ArgumentObject().AsString(field_EmailSubject))
		skbSendMail.PutString(sys.Storage_SendMail_Field_To, svCDocInvite.AsString(field_Email))
		skbSendMail.PutString(sys.Storage_SendMail_Field_Body, replacer.Replace(emailTemplate))
		skbSendMail.PutString(sys.Storage_SendMail_Field_From, smtpCfg.GetFrom())
		skbSendMail.PutString(sys.Storage_SendMail_Field_Host, smtpCfg.Host)
		skbSendMail.PutInt32(sys.Storage_SendMail_Field_Port, smtpCfg.Port)
		skbSendMail.PutString(sys.Storage_SendMail_Field_Username, smtpCfg.Username)

		pwd, err := state.ReadSecret(s, smtpCfg.PwdSecret)
		if err != nil {
			return err
		}
		skbSendMail.PutString(sys.Storage_SendMail_Field_Password, pwd)

		_, err = intents.NewValue(skbSendMail)
		if err != nil {
			return
		}

		//Update invite
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"Updated":%d,"Roles":"%s"}}]}`, event.ArgumentObject().AsRecordID(field_InviteID), State_Joined, time.Now().UnixMilli(), event.ArgumentObject().AsString(Field_Roles)),
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())

		return err
	}
}
