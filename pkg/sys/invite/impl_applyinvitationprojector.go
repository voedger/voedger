/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideAsyncProjectorApplyInvitationFactory(timeFunc coreutils.TimeFunc, federation coreutils.IFederation, appQName istructs.AppQName, tokens itokens.ITokens, smtpCfg smtp.Cfg) istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: qNameAPApplyInvitation,
			Func: applyInvitationProjector(timeFunc, federation, appQName, tokens, smtpCfg),
		}
	}
}

func applyInvitationProjector(timeFunc coreutils.TimeFunc, federation coreutils.IFederation, appQName istructs.AppQName, tokens itokens.ITokens, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		skbViewInviteIndex, err := s.KeyBuilder(state.View, qNameViewInviteIndex)
		if err != nil {
			return
		}
		skbViewInviteIndex.PutInt32(field_Dummy, value_Dummy_One)
		skbViewInviteIndex.PutString(Field_Login, event.ArgumentObject().AsString(field_Email))
		svViewInviteIndex, err := s.MustExist(skbViewInviteIndex)
		if err != nil {
			return
		}

		verificationCode, err := coreutils.EmailVerificationCode()
		if err != nil {
			// notest
			return err
		}
		emailTemplate := coreutils.TruncateEmailTemplate(event.ArgumentObject().AsString(field_EmailTemplate))

		skbCDocWorkspaceDescriptor, err := s.KeyBuilder(state.Record, authnz.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return
		}
		skbCDocWorkspaceDescriptor.PutQName(state.Field_Singleton, authnz.QNameCDocWorkspaceDescriptor)
		svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
		if err != nil {
			return
		}

		replacer := strings.NewReplacer(
			EmailTemplatePlaceholder_VerificationCode, verificationCode,
			EmailTemplatePlaceholder_InviteID, strconv.FormatInt(int64(svViewInviteIndex.AsRecordID(field_InviteID)), base),
			EmailTemplatePlaceholder_WSID, strconv.FormatInt(int64(event.Workspace()), base),
			EmailTemplatePlaceholder_WSName, svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName),
			EmailTemplatePlaceholder_Email, event.ArgumentObject().AsString(field_Email),
		)

		//Send invitation email
		skbSendMail, err := s.KeyBuilder(state.SendMail, appdef.NullQName)
		if err != nil {
			return
		}
		skbSendMail.PutString(state.Field_Subject, event.ArgumentObject().AsString(field_EmailSubject))
		skbSendMail.PutString(state.Field_To, event.ArgumentObject().AsString(field_Email))
		skbSendMail.PutString(state.Field_Body, replacer.Replace(emailTemplate))
		skbSendMail.PutString(state.Field_From, smtpCfg.GetFrom())
		skbSendMail.PutString(state.Field_Host, smtpCfg.Host)
		skbSendMail.PutInt32(state.Field_Port, smtpCfg.Port)
		skbSendMail.PutString(state.Field_Username, smtpCfg.Username)

		pwd := ""
		if !coreutils.IsTest() {
			skbAppSecretsStorage, err := s.KeyBuilder(state.AppSecret, appdef.NullQName)
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

		//Update invite status
		authToken, err := payloads.GetSystemPrincipalToken(tokens, appQName)
		if err != nil {
			return
		}
		_, err = coreutils.FederationFunc(
			federation.URL(),
			fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, event.Workspace()),
			fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d,"VerificationCode":"%s","Updated":%d}}]}`, svViewInviteIndex.AsRecordID(field_InviteID), State_Invited, verificationCode, timeFunc().UnixMilli()),
			coreutils.WithAuthorizeBy(authToken),
			coreutils.WithDiscardResponse())

		return err
	}
}
