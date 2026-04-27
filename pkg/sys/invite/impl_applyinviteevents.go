/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package invite

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/strconvu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

var projectorValidStates = map[appdef.QName]map[State]bool{
	qNameCmdInitiateInvitationByEMail:    {State_ToBeInvited: true},
	qNameCmdInitiateJoinWorkspace:        {State_Invited: true, State_ToBeJoined: true},
	qNameCmdInitiateUpdateInviteRoles:    {State_Joined: true, State_ToUpdateRoles: true},
	qNameCmdInitiateCancelAcceptedInvite: {State_Joined: true, State_ToBeCancelled: true, State_ToUpdateRoles: true},
	qNameCmdInitiateLeaveWorkspace:       {State_Joined: true, State_ToBeLeft: true, State_ToUpdateRoles: true},
	qNameCmdCancelSentInvite:             {State_ToBeInvited: true, State_Invited: true, State_ToBeJoined: true},
}

func asyncProjectorApplyInviteEvents(time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) istructs.Projector {
	return istructs.Projector{
		Name: qNameAPApplyInviteEvents,
		Func: applyInviteEvents(time, fed, tokens, smtpCfg),
	}
}

func applyInviteEvents(time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		cmd := event.QName()
		inviteID, err := inviteIDFromEvent(cmd, event, s)
		if err != nil {
			return err
		}
		if inviteID == istructs.NullRecordID {
			return nil
		}
		svCDocInvite, err := loadInviteByID(s, inviteID)
		if err != nil {
			return err
		}
		if !projectorValidStates[cmd][State(svCDocInvite.AsInt32(Field_State))] {
			return nil
		}
		switch cmd {
		case qNameCmdInitiateInvitationByEMail:
			return handleApplyInvitation(event, s, intents, inviteID, time, fed, tokens, smtpCfg)
		case qNameCmdInitiateJoinWorkspace:
			return handleApplyJoinWorkspace(event, s, svCDocInvite, inviteID, time, fed, tokens)
		case qNameCmdInitiateUpdateInviteRoles:
			return handleApplyUpdateInviteRoles(event, s, intents, svCDocInvite, inviteID, time, fed, tokens, smtpCfg)
		case qNameCmdInitiateCancelAcceptedInvite:
			return handleApplyCancelAcceptedInvite(event, s, svCDocInvite, inviteID, time, fed, tokens)
		case qNameCmdInitiateLeaveWorkspace:
			return handleApplyLeaveWorkspace(event, s, svCDocInvite, inviteID, time, fed, tokens)
		case qNameCmdCancelSentInvite:
			return handleCancelSentInvite(event, s, inviteID, time, fed, tokens)
		}
		return nil
	}
}

func inviteIDFromEvent(cmd appdef.QName, event istructs.IPLogEvent, s istructs.IState) (istructs.RecordID, error) {
	switch cmd {
	case qNameCmdInitiateInvitationByEMail:
		skb, err := s.KeyBuilder(sys.Storage_View, qNameViewInviteIndex)
		if err != nil {
			return istructs.NullRecordID, err
		}
		skb.PutInt32(field_Dummy, value_Dummy_One)
		skb.PutString(Field_Login, event.ArgumentObject().AsString(Field_Email))
		sv, err := s.MustExist(skb)
		if err != nil {
			return istructs.NullRecordID, err
		}
		return sv.AsRecordID(field_InviteID), nil
	case qNameCmdInitiateLeaveWorkspace:
		for rec := range event.CUDs {
			if rec.QName() == QNameCDocInvite {
				return rec.ID(), nil
			}
		}
		return istructs.NullRecordID, nil
	default:
		return event.ArgumentObject().AsRecordID(field_InviteID), nil
	}
}

func loadInviteByID(s istructs.IState, inviteID istructs.RecordID) (istructs.IStateValue, error) {
	skb, err := s.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
	if err != nil {
		return nil, err
	}
	skb.PutRecordID(sys.Storage_Record_Field_ID, inviteID)
	return s.MustExist(skb)
}

func getSystemToken(s istructs.IState, tokens itokens.ITokens) (string, appdef.AppQName, error) {
	appQName := s.App()
	token, err := payloads.GetSystemPrincipalToken(tokens, appQName)
	return token, appQName, err
}

func cudURL(appQName appdef.AppQName, wsid istructs.WSID) string {
	return fmt.Sprintf("api/%s/%d/c.sys.CUD", appQName, wsid)
}

func updateInviteViaCUD(fed federation.IFederation, appQName appdef.AppQName, wsid istructs.WSID, token string, inviteID istructs.RecordID, fields string) error {
	_, err := fed.Func(
		cudURL(appQName, wsid),
		fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{%s}}]}`, inviteID, fields),
		httpu.WithAuthorizeBy(token),
		httpu.WithDiscardResponse())
	return err
}

func deactivateSubjectAndJoinedWorkspace(fed federation.IFederation, appQName appdef.AppQName, wsid istructs.WSID, token string, svCDocInvite istructs.IStateValue) error {
	subjectID := svCDocInvite.AsRecordID(field_SubjectID)
	_, err := fed.Func(
		cudURL(appQName, wsid),
		fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, subjectID),
		httpu.WithAuthorizeBy(token),
		httpu.WithDiscardResponse())
	if err != nil {
		return err
	}
	_, err = fed.Func(
		fmt.Sprintf("api/%s/%d/c.sys.DeactivateJoinedWorkspace", appQName, svCDocInvite.AsInt64(Field_InviteeProfileWSID)),
		fmt.Sprintf(`{"args":{"InvitingWorkspaceWSID":%d}}`, wsid),
		httpu.WithAuthorizeBy(token),
		httpu.WithDiscardResponse())
	return err
}

func handleApplyInvitation(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents, inviteID istructs.RecordID, time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) error {
	verificationCode := coreutils.EmailVerificationCode()
	emailTemplate := coreutils.TruncateEmailTemplate(event.ArgumentObject().AsString(field_EmailTemplate))

	skbCDocWorkspaceDescriptor, err := s.KeyBuilder(sys.Storage_Record, appdef.QNameCDocWorkspaceDescriptor)
	if err != nil {
		return err
	}
	skbCDocWorkspaceDescriptor.PutQName(sys.Storage_Record_Field_Singleton, appdef.QNameCDocWorkspaceDescriptor)
	svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
	if err != nil {
		return err
	}

	replacer := strings.NewReplacer(
		EmailTemplatePlaceholder_VerificationCode, verificationCode,
		EmailTemplatePlaceholder_InviteID, strconvu.UintToString(inviteID),
		EmailTemplatePlaceholder_WSID, strconvu.UintToString(event.Workspace()),
		EmailTemplatePlaceholder_WSName, svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName),
		EmailTemplatePlaceholder_Email, event.ArgumentObject().AsString(Field_Email),
	)

	if err = sendEmail(s, intents, smtpCfg,
		event.ArgumentObject().AsString(field_EmailSubject),
		event.ArgumentObject().AsString(Field_Email),
		replacer.Replace(emailTemplate)); err != nil {
		return err
	}

	token, appQName, err := getSystemToken(s, tokens)
	if err != nil {
		return err
	}
	return updateInviteViaCUD(fed, appQName, event.Workspace(), token, inviteID,
		fmt.Sprintf(`"State":%d,"VerificationCode":"%s","Updated":%d`, State_Invited, verificationCode, time.Now().UnixMilli()))
}

func sendEmail(s istructs.IState, intents istructs.IIntents, smtpCfg smtp.Cfg, subject, to, body string) error {
	skbSendMail, err := s.KeyBuilder(sys.Storage_SendMail, appdef.NullQName)
	if err != nil {
		return err
	}
	skbSendMail.PutString(sys.Storage_SendMail_Field_Subject, subject)
	skbSendMail.PutString(sys.Storage_SendMail_Field_To, to)
	skbSendMail.PutString(sys.Storage_SendMail_Field_Body, body)
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
	return err
}

func handleApplyJoinWorkspace(event istructs.IPLogEvent, s istructs.IState, svCDocInvite istructs.IStateValue, inviteID istructs.RecordID, time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens) error {
	login := svCDocInvite.AsString(field_ActualLogin)
	existingSubjectID, isActive, err := SubjectExistsByLogin(login, s)
	if err != nil {
		return err
	}

	skbCDocWorkspaceDescriptor, err := s.KeyBuilder(sys.Storage_Record, appdef.QNameCDocWorkspaceDescriptor)
	if err != nil {
		return err
	}
	skbCDocWorkspaceDescriptor.PutQName(sys.Storage_Record_Field_Singleton, appdef.QNameCDocWorkspaceDescriptor)
	svCDocWorkspaceDescriptor, err := s.MustExist(skbCDocWorkspaceDescriptor)
	if err != nil {
		return err
	}

	token, appQName, err := getSystemToken(s, tokens)
	if err != nil {
		return err
	}

	_, err = fed.Func(
		fmt.Sprintf("api/%s/%d/c.sys.CreateJoinedWorkspace", appQName, svCDocInvite.AsInt64(Field_InviteeProfileWSID)),
		fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d,"WSName":%q}}`,
			svCDocInvite.AsString(Field_Roles), event.Workspace(), svCDocWorkspaceDescriptor.AsString(authnz.Field_WSName)),
		httpu.WithAuthorizeBy(token),
		httpu.WithDiscardResponse(),
	)
	if err != nil {
		return err
	}

	var body string
	switch {
	case existingSubjectID == istructs.NullRecordID:
		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
			svCDocInvite.AsString(field_ActualLogin), svCDocInvite.AsString(Field_Roles), svCDocInvite.AsInt32(authnz.Field_SubjectKind),
			svCDocInvite.AsInt64(Field_InviteeProfileWSID))
	case !isActive:
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, existingSubjectID)
		_, err = fed.Func(
			cudURL(appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())
		if err != nil {
			return err
		}
		fallthrough
	default:
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, existingSubjectID, svCDocInvite.AsString(Field_Roles))
	}
	subjectID := existingSubjectID
	if existingSubjectID == istructs.NullRecordID {
		resp, err := fed.Func(
			cudURL(appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token))
		if err != nil {
			return err
		}
		subjectID = resp.NewID()
	} else {
		_, err = fed.Func(
			cudURL(appQName, event.Workspace()),
			body,
			httpu.WithAuthorizeBy(token),
			httpu.WithDiscardResponse())
		if err != nil {
			return err
		}
	}
	return updateInviteViaCUD(fed, appQName, event.Workspace(), token, inviteID,
		fmt.Sprintf(`"State":%d,"SubjectID":%d,"Updated":%d`, State_Joined, subjectID, time.Now().UnixMilli()))
}

func handleApplyUpdateInviteRoles(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents, svCDocInvite istructs.IStateValue, inviteID istructs.RecordID, time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens, smtpCfg smtp.Cfg) error {
	token, appQName, err := getSystemToken(s, tokens)
	if err != nil {
		return err
	}

	subjectID := svCDocInvite.AsRecordID(field_SubjectID)
	_, err = fed.Func(
		cudURL(appQName, event.Workspace()),
		fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Roles":"%s"}}]}`, subjectID, event.ArgumentObject().AsString(Field_Roles)),
		httpu.WithAuthorizeBy(token),
		httpu.WithDiscardResponse())
	if err != nil {
		return err
	}

	_, err = fed.Func(
		fmt.Sprintf("api/%s/%d/c.sys.UpdateJoinedWorkspaceRoles", appQName, svCDocInvite.AsInt64(Field_InviteeProfileWSID)),
		fmt.Sprintf(`{"args":{"Roles":"%s","InvitingWorkspaceWSID":%d}}`, event.ArgumentObject().AsString(Field_Roles), event.Workspace()),
		httpu.WithAuthorizeBy(token),
		httpu.WithDiscardResponse())
	if err != nil {
		return err
	}

	emailTemplate := coreutils.TruncateEmailTemplate(event.ArgumentObject().AsString(field_EmailTemplate))
	replacer := strings.NewReplacer(EmailTemplatePlaceholder_Roles, event.ArgumentObject().AsString(Field_Roles))

	if err = sendEmail(s, intents, smtpCfg,
		event.ArgumentObject().AsString(field_EmailSubject),
		svCDocInvite.AsString(Field_Email),
		replacer.Replace(emailTemplate)); err != nil {
		return err
	}

	return updateInviteViaCUD(fed, appQName, event.Workspace(), token, inviteID,
		fmt.Sprintf(`"State":%d,"Updated":%d,"Roles":"%s"`, State_Joined, time.Now().UnixMilli(), event.ArgumentObject().AsString(Field_Roles)))
}

func handleApplyCancelAcceptedInvite(event istructs.IPLogEvent, s istructs.IState, svCDocInvite istructs.IStateValue, inviteID istructs.RecordID, time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens) error {
	token, appQName, err := getSystemToken(s, tokens)
	if err != nil {
		return err
	}

	if err = deactivateSubjectAndJoinedWorkspace(fed, appQName, event.Workspace(), token, svCDocInvite); err != nil {
		return err
	}
	return updateInviteViaCUD(fed, appQName, event.Workspace(), token, inviteID,
		fmt.Sprintf(`"State":%d,"Updated":%d`, State_Cancelled, time.Now().UnixMilli()))
}

func handleApplyLeaveWorkspace(event istructs.IPLogEvent, s istructs.IState, svCDocInvite istructs.IStateValue, inviteID istructs.RecordID, time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens) error {
	token, appQName, err := getSystemToken(s, tokens)
	if err != nil {
		return err
	}

	if err = deactivateSubjectAndJoinedWorkspace(fed, appQName, event.Workspace(), token, svCDocInvite); err != nil {
		return err
	}
	return updateInviteViaCUD(fed, appQName, event.Workspace(), token, inviteID,
		fmt.Sprintf(`"State":%d,"Updated":%d`, State_Left, time.Now().UnixMilli()))
}

func handleCancelSentInvite(event istructs.IPLogEvent, s istructs.IState, inviteID istructs.RecordID, time timeu.ITime, fed federation.IFederation, tokens itokens.ITokens) error {
	token, appQName, err := getSystemToken(s, tokens)
	if err != nil {
		return err
	}

	return updateInviteViaCUD(fed, appQName, event.Workspace(), token, inviteID,
		fmt.Sprintf(`"State":%d,"Updated":%d`, State_Cancelled, time.Now().UnixMilli()))
}
