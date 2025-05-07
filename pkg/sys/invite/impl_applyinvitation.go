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
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func asyncProjectorApplyInvitation(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyInvitation,
		Func: applyInvitationProjector(time, federation, tokens, smtpCfg),
	}
}

// AFTER EXECUTE ON (InitiateInvitationByEMail)
func applyInvitationProjector(time timeu.ITime, federation federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		skbViewInviteIndex, err := s.KeyBuilder(sys.Storage_View, qNameViewInviteIndex)
		if err != nil {
			return
		}
		skbViewInviteIndex.PutInt32(field_Dummy, value_Dummy_One)
		skbViewInviteIndex.PutString(Field_Login, event.ArgumentObject().AsString(field_Email))
		svViewInviteIndex, err := s.MustExist(skbViewInviteIndex)
		if err != nil {
			return
		}

		verificationCode := coreutils.EmailVerificationCode()
		emailTemplate := coreutils.TruncateEmailTemplate(event.ArgumentObject().AsString(field_EmailTemplate))

		skbCDocWorkspaceDescriptor, err := s.KeyBuilder(sys.Storage_Record, authnz.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return
		}
		skbCDocWorkspaceDescriptor.PutQName(sys.Storage_Record_Field_Singleton, authnz.QNameCDocWorkspaceDescriptor)
		svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
		if err != nil {
			return
		}

		replacer := strings.NewReplacer(
			EmailTemplatePlaceholder_VerificationCode, verificationCode,
			EmailTemplatePlaceholder_InviteID, utils.UintToString(svViewInviteIndex.AsRecordID(field_InviteID)),
			EmailTemplatePlaceholder_WSID, utils.UintToString(event.Workspace()),
			EmailTemplatePlaceholder_WSName, svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName),
			EmailTemplatePlaceholder_Email, event.ArgumentObject().AsString(field_Email),
		)

		// Send invitation email
		skbSendMail, err := s.KeyBuilder(sys.Storage_SendMail, appdef.NullQName)
		if err != nil {
			return
		}
		skbSendMail.PutString(sys.Storage_SendMail_Field_Subject, event.ArgumentObject().AsString(field_EmailSubject))
		skbSendMail.PutString(sys.Storage_SendMail_Field_To, event.ArgumentObject().AsString(field_Email))
		skbSendMail.PutString(sys.Storage_SendMail_Field_Body, replacer.Replace(emailTemplate))
		skbSendMail.PutString(sys.Storage_SendMail_Field_From, smtpCfg.GetFrom())
		skbSendMail.PutString(sys.Storage_SendMail_Field_Host, smtpCfg.Host)
		skbSendMail.PutInt32(sys.Storage_SendMail_Field_Port, smtpCfg.Port)
		skbSendMail.PutString(sys.Storage_SendMail_Field_Username, smtpCfg.Username)

		pwd := ""
		if !coreutils.IsTest() {
			skbAppSecretsStorage, err := s.KeyBuilder(sys.Storage_AppSecret, appdef.NullQName)
			if err != nil {
				return err
			}
			skbAppSecretsStorage.PutString(sys.Storage_AppSecretField_Secret, smtpCfg.PwdSecret)
			svAppSecretsStorage, err := s.MustExist(skbAppSecretsStorage)
			if err != nil {
				return err
			}
			pwd = svAppSecretsStorage.AsString("")
		}
		skbSendMail.PutString(sys.Storage_SendMail_Field_Password, pwd)

		// Send invitation Email
		_, err = intents.NewValue(skbSendMail)
		if err != nil {
			return
		}

		// Update cdoc.Invite State=Invited
		appQName := s.App()
		authToken, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}
		_, err = federation.Func(
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"VerificationCode":"%s","Updated":%d}}]}`, svViewInviteIndex.AsRecordID(field_InviteID), State_Invited, verificationCode, time.Now().UnixMilli()),
			coreutils.WithAuthorizeBy(authToken),
			coreutils.WithDiscardResponse())

		return err
	}
}
