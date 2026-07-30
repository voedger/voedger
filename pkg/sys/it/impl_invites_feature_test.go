/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/invite"
	it "github.com/voedger/voedger/pkg/vit"
)

type invitesFeatureFixture struct {
	vit       *it.VIT
	ws        *it.AppWorkspace
	login     it.Login
	principal *it.Principal
	email     string
}

type invitesFeatureInvite struct {
	email              string
	roles              string
	state              invite.State
	subjectID          istructs.RecordID
	actualLogin        string
	inviteeProfileWSID istructs.WSID
	subjectKind        istructs.SubjectKindType
}

type invitesFeatureSubject struct {
	id       istructs.RecordID
	login    string
	roles    string
	isActive bool
}

// [~server.invites.invite/it~impl]
func TestInvites(t *testing.T) {
	t.Run("invites: scn: Workspace owner sends an invitation", func(t *testing.T) {
		// Given Workspace "Acme" exists
		// And User Login "alice@example.com" exists
		f := newInvitesFeatureFixture(t, "send")

		// When Workspace Owner invites "alice@example.com" to Workspace "Acme" with Role "app1pkg.LimitedAccessRole"
		inviteID, message, _ := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// Then "alice@example.com" receives an invitation email for Workspace "Acme"
		require.Equal(t, []string{f.email}, message.To)
		messageFields := invitationMessageFields(message)
		require.Len(t, messageFields, 5)
		require.Equal(t, fmt.Sprint(inviteID), messageFields[1])
		require.Equal(t, fmt.Sprint(f.ws.WSID), messageFields[2])
		require.Equal(t, f.ws.Name, messageFields[3])
		require.Equal(t, f.email, messageFields[4])

		// And Workspace "Acme" has a pending invitation for "alice@example.com" with Role "app1pkg.LimitedAccessRole"
		invitation := getInvitesFeatureInvitation(t, f, inviteID)
		require.Equal(t, f.email, invitation.email)
		require.Equal(t, initialRoles, invitation.roles)
		require.Equal(t, invite.State_Invited, invitation.state)
	})

	t.Run("invites: scn: Workspace owner resends a pending invitation", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "resend")

		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		inviteID, _, firstCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When Workspace Owner resends the invitation
		message, secondCode := resendInvitesFeatureInvitation(t, f, inviteID, f.email, initialRoles)

		// Then "alice@example.com" receives a new invitation verification code
		require.Equal(t, []string{f.email}, message.To)
		require.NotEqual(t, firstCode, secondCode)

		// And the pending invitation remains for Workspace "Acme"
		invitation := getInvitesFeatureInvitation(t, f, inviteID)
		require.Equal(t, f.email, invitation.email)
		require.Equal(t, invite.State_Invited, invitation.state)
	})

	t.Run("invites: scn: Workspace owner changes roles while resending a pending invitation", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "resend_roles")

		// Given Workspace "Acme" has a pending invitation for "alice@example.com" with Role "app1pkg.LimitedAccessRole"
		inviteID, _, _ := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When Workspace Owner resends the invitation with Role "app1pkg.SpecialAPITokenRole"
		resendInvitesFeatureInvitation(t, f, inviteID, f.email, newRoles)

		// Then the pending invitation has Role "app1pkg.SpecialAPITokenRole"
		require.Equal(t, newRoles, getInvitesFeatureInvitation(t, f, inviteID).roles)
	})

	t.Run("invites: scn: Workspace owner cancels a pending invitation", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "cancel_pending")

		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When Workspace Owner cancels the invitation
		f.vit.PostWS(f.ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Invited, invite.State_Cancelled)

		// Then the response status is "400 Bad Request" when User Login "alice@example.com" tries to accept it
		InitiateJoinWorkspace(f.vit, f.ws, inviteID, f.principal, verificationCode, httpu.Expect400())

		// And User Login "alice@example.com" is not a member of Workspace "Acme"
		require.Empty(t, getInvitesFeatureSubjects(t, f, f.email))
	})

	t.Run("invites: scn: Workspace owner cannot cancel a non-existing invitation", func(t *testing.T) {
		// Given Workspace "Acme" has no invitation with ID "66048"
		f := newInvitesFeatureFixture(t, "cancel_non_existing")

		// When Workspace Owner cancels invitation with ID "66048"
		// Then the response status is "400 Bad Request"
		// And the response reports that the invitation does not exist
		f.vit.PostWS(f.ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, istructs.NonExistingRecordID), it.Expect400RefIntegrity_Existence())
	})

	t.Run("invites: scn: Workspace owner reinvites after cancelling a pending invitation", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "reinvite_cancelled")

		// Given Workspace Owner cancelled a pending invitation for "alice@example.com" to Workspace "Acme"
		inviteID, _, firstCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		f.vit.PostWS(f.ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Invited, invite.State_Cancelled)

		// When Workspace Owner reinvites "alice@example.com"
		_, secondCode := resendInvitesFeatureInvitation(t, f, inviteID, f.email, initialRoles)

		// Then "alice@example.com" receives a new invitation verification code
		require.NotEqual(t, firstCode, secondCode)

		// And Workspace "Acme" has a pending invitation for "alice@example.com"
		require.Equal(t, invite.State_Invited, getInvitesFeatureInvitation(t, f, inviteID).state)
	})

	t.Run("invites: scn: Workspace owner cannot invite an existing member", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "invite_member")

		// Given User Login "alice@example.com" is a member of Workspace "Acme"
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, verificationCode)

		// When Workspace Owner invites "alice@example.com" to Workspace "Acme"
		// Then the response status is "400 Bad Request"
		body := fmt.Sprintf(`{"args":{"Email":"%s","Roles":"%s","ExpireDatetime":%d,"EmailTemplate":"%s","EmailSubject":"%s"}}`,
			f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli(), inviteEmailTemplate, inviteEmailSubject)
		f.vit.PostWS(f.ws, "c.sys.InitiateInvitationByEMail", body, httpu.Expect400())
	})

	t.Run("invites: scn: User accepts an invitation addressed to an authenticated identifier: canonical login", func(t *testing.T) {
		// | recipient           |
		// | jsmith@example.com  |
		f := newInvitesFeatureFixture(t, "accept_canonical")

		// Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
		principal, _ := setInvitesFeatureAlias(t, f)

		// And Workspace "Acme" has an invitation for "<recipient>"
		// recipient = jsmith@example.com
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When User Login "jsmith@example.com" submits the invitation verification code
		acceptInvitesFeatureInvitation(t, f, inviteID, principal, verificationCode)

		// Then User Login "jsmith@example.com" becomes a member of Workspace "Acme"
		subject := requireInvitesFeatureMembership(t, f, f.email, true)
		require.Equal(t, initialRoles, subject.roles)

		// And the membership identifies canonical User Login "jsmith@example.com"
		require.Equal(t, f.email, subject.login)
		invitation := getInvitesFeatureInvitation(t, f, inviteID)
		require.Equal(t, f.email, invitation.actualLogin)
		require.Equal(t, f.principal.ProfileWSID, invitation.inviteeProfileWSID)
		require.Equal(t, istructs.SubjectKind_User, invitation.subjectKind)
		require.NotEqual(t, istructs.NullRecordID, invitation.subjectID)
		joinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(f.vit, f.ws.WSID, f.principal)
		require.Equal(t, initialRoles, joinedWorkspace.roles)
		require.Equal(t, f.ws.Name, joinedWorkspace.wsName)
	})

	t.Run("invites: scn: User accepts an invitation addressed to an authenticated identifier: active alias", func(t *testing.T) {
		// | recipient           |
		// | j.smith@example.com |
		f := newInvitesFeatureFixture(t, "accept_alias")

		// Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
		principal, alias := setInvitesFeatureAlias(t, f)

		// And Workspace "Acme" has an invitation for "<recipient>"
		// recipient = j.smith@example.com
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, alias, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When User Login "jsmith@example.com" submits the invitation verification code
		acceptInvitesFeatureInvitation(t, f, inviteID, principal, verificationCode)

		// Then User Login "jsmith@example.com" becomes a member of Workspace "Acme"
		subject := requireInvitesFeatureMembership(t, f, f.email, true)
		require.Equal(t, initialRoles, subject.roles)

		// And the membership identifies canonical User Login "jsmith@example.com"
		require.Equal(t, f.email, subject.login)
		require.Equal(t, f.email, getInvitesFeatureInvitation(t, f, inviteID).actualLogin)
	})

	t.Run("invites: scn: User cannot accept an invitation addressed to another identity", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "reject_identity")

		// Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
		principal, _ := setInvitesFeatureAlias(t, f)

		// And Workspace "Acme" has an invitation for "other@example.com"
		otherEmail := fmt.Sprintf("other_%d@example.com", f.vit.NextNumber())
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, otherEmail, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When User Login "jsmith@example.com" submits the invitation verification code
		// Then the response status is "400 Bad Request"
		InitiateJoinWorkspace(f.vit, f.ws, inviteID, principal, verificationCode, httpu.Expect400())

		// And User Login "jsmith@example.com" is not a member of Workspace "Acme"
		require.Empty(t, getInvitesFeatureSubjects(t, f, f.email))
	})

	t.Run("invites: scn: User cannot accept an unusable invitation: expired", func(t *testing.T) {
		// | condition                         |
		// | is expired                        |
		f := newInvitesFeatureFixture(t, "reject_expired")

		// Given Workspace "Acme" has an invitation for User Login "alice@example.com" that <condition>
		// condition = is expired
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(-time.Millisecond).UnixMilli())

		// When User Login "alice@example.com" submits the invitation verification code
		// Then the response status is "400 Bad Request"
		InitiateJoinWorkspace(f.vit, f.ws, inviteID, f.principal, verificationCode, httpu.Expect400())

		// And User Login "alice@example.com" is not a member of Workspace "Acme"
		require.Empty(t, getInvitesFeatureSubjects(t, f, f.email))
	})

	t.Run("invites: scn: User cannot accept an unusable invitation: different verification code", func(t *testing.T) {
		// | condition                         |
		// | has a different verification code |
		f := newInvitesFeatureFixture(t, "reject_code")

		// Given Workspace "Acme" has an invitation for User Login "alice@example.com" that <condition>
		// condition = has a different verification code
		inviteID, _, _ := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When User Login "alice@example.com" submits the invitation verification code
		// Then the response status is "400 Bad Request"
		InitiateJoinWorkspace(f.vit, f.ws, inviteID, f.principal, "wrong-code", httpu.Expect400())

		// And User Login "alice@example.com" is not a member of Workspace "Acme"
		require.Empty(t, getInvitesFeatureSubjects(t, f, f.email))
	})

	t.Run("invites: scn: User cannot accept an unusable invitation: cancelled", func(t *testing.T) {
		// | condition                         |
		// | was cancelled                     |
		f := newInvitesFeatureFixture(t, "reject_cancelled")

		// Given Workspace "Acme" has an invitation for User Login "alice@example.com" that <condition>
		// condition = was cancelled
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		f.vit.PostWS(f.ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Invited, invite.State_Cancelled)

		// When User Login "alice@example.com" submits the invitation verification code
		// Then the response status is "400 Bad Request"
		InitiateJoinWorkspace(f.vit, f.ws, inviteID, f.principal, verificationCode, httpu.Expect400())

		// And User Login "alice@example.com" is not a member of Workspace "Acme"
		require.Empty(t, getInvitesFeatureSubjects(t, f, f.email))
	})

	t.Run("invites: scn: Alias-addressed invitation reuses an existing canonical membership", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "alias_reuse")

		// Given User Login "jsmith@example.com" has active Login Alias "j.smith@example.com"
		principal, alias := setInvitesFeatureAlias(t, f)

		// And User Login "jsmith@example.com" is already a member of Workspace "Acme"
		canonicalInviteID, _, canonicalCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, canonicalInviteID, principal, canonicalCode)

		// And Workspace "Acme" has an invitation for "j.smith@example.com"
		aliasInviteID, _, aliasCode := sendInvitesFeatureInvitation(t, f, alias, newRoles, f.vit.Now().Add(time.Hour).UnixMilli())

		// When User Login "jsmith@example.com" submits the invitation verification code
		acceptInvitesFeatureInvitation(t, f, aliasInviteID, principal, aliasCode)

		// Then Workspace "Acme" has exactly one membership for User Login "jsmith@example.com"
		subjects := getInvitesFeatureSubjects(t, f, f.email)
		require.Len(t, subjects, 1)
		require.True(t, subjects[0].isActive)
		require.Equal(t, newRoles, subjects[0].roles)
		require.Equal(t, f.email, getInvitesFeatureInvitation(t, f, aliasInviteID).actualLogin)
	})

	t.Run("invites: scn: Workspace owner updates an invited member's roles", func(t *testing.T) {
		f := newInvitesFeatureFixture(t, "update_roles")

		// Given User Login "alice@example.com" joined Workspace "Acme" with Role "app1pkg.LimitedAccessRole"
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, verificationCode)

		// When Workspace Owner updates the membership to Role "app1pkg.SpecialAPITokenRole"
		body := fmt.Sprintf(`{"args":{"InviteID":%d,"Roles":"%s","EmailTemplate":"%s","EmailSubject":"%s"}}`,
			inviteID, newRoles, "text:"+invite.EmailTemplatePlaceholder_Roles, "roles updated")
		f.vit.PostWS(f.ws, "c.sys.InitiateUpdateInviteRoles", body)
		message := f.vit.CaptureEmail()

		// Then User Login "alice@example.com" has Role "app1pkg.SpecialAPITokenRole" in Workspace "Acme"
		subject := requireInvitesFeatureMembership(t, f, f.email, true)
		require.Equal(t, newRoles, subject.roles)

		// And the user's joined-workspace record has Role "app1pkg.SpecialAPITokenRole"
		joinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(f.vit, f.ws.WSID, f.principal)
		require.Equal(t, newRoles, joinedWorkspace.roles)

		// And "alice@example.com" receives a role-update email
		require.Equal(t, []string{f.email}, message.To)
		require.Equal(t, "roles updated", message.Subject)
		require.Equal(t, it.TestSMTPCfg.GetFrom(), message.From)
		require.Equal(t, newRoles, message.Body)
	})

	t.Run("invites: scn: Workspace membership ends: workspace owner removes member", func(t *testing.T) {
		// | action                                                           |
		// | Workspace Owner removes User Login "alice@example.com"           |
		f := newInvitesFeatureFixture(t, "owner_removes")

		// Given User Login "alice@example.com" is a member of Workspace "Acme"
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, verificationCode)

		// When <action>
		// action = Workspace Owner removes User Login "alice@example.com"
		f.vit.PostWS(f.ws, "c.sys.InitiateCancelAcceptedInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Joined, invite.State_Cancelled)

		// Then User Login "alice@example.com" is not an active member of Workspace "Acme"
		requireInvitesFeatureMembership(t, f, f.email, false)

		// And the user's joined-workspace record is inactive
		require.False(t, FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(f.vit, f.ws.WSID, f.principal).isActive)
	})

	t.Run("invites: scn: Workspace membership ends: member leaves", func(t *testing.T) {
		// | action                                                           |
		// | User Login "alice@example.com" leaves Workspace "Acme"           |
		f := newInvitesFeatureFixture(t, "member_leaves")

		// Given User Login "alice@example.com" is a member of Workspace "Acme"
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, verificationCode)

		// When <action>
		// action = User Login "alice@example.com" leaves Workspace "Acme"
		f.vit.PostWS(f.ws, "c.sys.InitiateLeaveWorkspace", "{}", httpu.WithAuthorizeBy(f.principal.Token))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Joined, invite.State_Left)

		// Then User Login "alice@example.com" is not an active member of Workspace "Acme"
		requireInvitesFeatureMembership(t, f, f.email, false)

		// And the user's joined-workspace record is inactive
		require.False(t, FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(f.vit, f.ws.WSID, f.principal).isActive)
	})

	t.Run("invites: scn: Previous member accepts a new invitation: removed member", func(t *testing.T) {
		// | membership end             |
		// | was removed from           |
		f := newInvitesFeatureFixture(t, "restore_removed")

		// Given User Login "alice@example.com" previously <membership end> Workspace "Acme"
		// membership end = was removed from
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, verificationCode)
		f.vit.PostWS(f.ws, "c.sys.InitiateCancelAcceptedInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Joined, invite.State_Cancelled)

		// When Workspace Owner reinvites User Login "alice@example.com"
		_, newCode := resendInvitesFeatureInvitation(t, f, inviteID, f.email, newRoles)

		// And User Login "alice@example.com" accepts the new invitation
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, newCode)

		// Then User Login "alice@example.com" is an active member of Workspace "Acme"
		subject := requireInvitesFeatureMembership(t, f, f.email, true)
		require.Equal(t, newRoles, subject.roles)

		// And Workspace "Acme" has exactly one membership for User Login "alice@example.com"
		require.Len(t, getInvitesFeatureSubjects(t, f, f.email), 1)
	})

	t.Run("invites: scn: Previous member accepts a new invitation: member who left", func(t *testing.T) {
		// | membership end             |
		// | left                       |
		f := newInvitesFeatureFixture(t, "restore_left")

		// Given User Login "alice@example.com" previously <membership end> Workspace "Acme"
		// membership end = left
		inviteID, _, verificationCode := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, verificationCode)
		f.vit.PostWS(f.ws, "c.sys.InitiateLeaveWorkspace", "{}", httpu.WithAuthorizeBy(f.principal.Token))
		WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Joined, invite.State_Left)

		// When Workspace Owner reinvites User Login "alice@example.com"
		_, newCode := resendInvitesFeatureInvitation(t, f, inviteID, f.email, newRoles)

		// And User Login "alice@example.com" accepts the new invitation
		acceptInvitesFeatureInvitation(t, f, inviteID, f.principal, newCode)

		// Then User Login "alice@example.com" is an active member of Workspace "Acme"
		subject := requireInvitesFeatureMembership(t, f, f.email, true)
		require.Equal(t, newRoles, subject.roles)

		// And Workspace "Acme" has exactly one membership for User Login "alice@example.com"
		require.Len(t, getInvitesFeatureSubjects(t, f, f.email), 1)
	})

	t.Run("invites: scn: Workspace owner cannot invite a malformed email address: missing at-sign", func(t *testing.T) {
		// | email |
		// | a     |
		f := newInvitesFeatureFixture(t, "invalid_email_a")

		// When Workspace Owner invites "<email>" to Workspace "Acme"
		// email = a
		// Then the response status is "400 Bad Request"
		postInvitesFeatureInvitation(t, f, "a", initialRoles, httpu.Expect400())
	})

	t.Run("invites: scn: Workspace owner cannot invite a malformed email address: missing domain", func(t *testing.T) {
		// | email |
		// | bad@  |
		f := newInvitesFeatureFixture(t, "invalid_email_domain")

		// When Workspace Owner invites "<email>" to Workspace "Acme"
		// email = bad@
		// Then the response status is "400 Bad Request"
		postInvitesFeatureInvitation(t, f, "bad@", initialRoles, httpu.Expect400())
	})

	t.Run("invites: scn: Workspace owner cannot invite a malformed email address: missing local part", func(t *testing.T) {
		// | email |
		// | @bad  |
		f := newInvitesFeatureFixture(t, "invalid_email_local")

		// When Workspace Owner invites "<email>" to Workspace "Acme"
		// email = @bad
		// Then the response status is "400 Bad Request"
		postInvitesFeatureInvitation(t, f, "@bad", initialRoles, httpu.Expect400())
	})

	t.Run("invites: scn: Workspace owner cannot send an invitation with an invalid role set: empty", func(t *testing.T) {
		// | roles                                                     |
		// |                                                           |
		// When Workspace Owner sends an invitation with Roles "<roles>"
		// roles =
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "", false)
	})

	t.Run("invites: scn: Workspace owner cannot send an invitation with an invalid role set: malformed QName", func(t *testing.T) {
		// | roles                                                     |
		// | not-a-qname                                               |
		// When Workspace Owner sends an invitation with Roles "<roles>"
		// roles = not-a-qname
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "not-a-qname", false)
	})

	t.Run("invites: scn: Workspace owner cannot send an invitation with an invalid role set: system role", func(t *testing.T) {
		// | roles                                                     |
		// | sys.WorkspaceOwner                                        |
		// When Workspace Owner sends an invitation with Roles "<roles>"
		// roles = sys.WorkspaceOwner
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "sys.WorkspaceOwner", false)
	})

	t.Run("invites: scn: Workspace owner cannot send an invitation with an invalid role set: missing role", func(t *testing.T) {
		// | roles                                                     |
		// | app1pkg.NonExistentRole                                   |
		// When Workspace Owner sends an invitation with Roles "<roles>"
		// roles = app1pkg.NonExistentRole
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "app1pkg.NonExistentRole", false)
	})

	t.Run("invites: scn: Workspace owner cannot send an invitation with an invalid role set: duplicate role", func(t *testing.T) {
		// | roles                                                     |
		// | app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole       |
		// When Workspace Owner sends an invitation with Roles "<roles>"
		// roles = app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole", false)
	})

	t.Run("invites: scn: Workspace owner cannot update an invitation with an invalid role set: empty", func(t *testing.T) {
		// | roles                                                     |
		// |                                                           |
		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		// When Workspace Owner updates the invitation with Roles "<roles>"
		// roles =
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "", true)
	})

	t.Run("invites: scn: Workspace owner cannot update an invitation with an invalid role set: malformed QName", func(t *testing.T) {
		// | roles                                                     |
		// | not-a-qname                                               |
		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		// When Workspace Owner updates the invitation with Roles "<roles>"
		// roles = not-a-qname
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "not-a-qname", true)
	})

	t.Run("invites: scn: Workspace owner cannot update an invitation with an invalid role set: system role", func(t *testing.T) {
		// | roles                                                     |
		// | sys.WorkspaceOwner                                        |
		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		// When Workspace Owner updates the invitation with Roles "<roles>"
		// roles = sys.WorkspaceOwner
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "sys.WorkspaceOwner", true)
	})

	t.Run("invites: scn: Workspace owner cannot update an invitation with an invalid role set: missing role", func(t *testing.T) {
		// | roles                                                     |
		// | app1pkg.NonExistentRole                                   |
		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		// When Workspace Owner updates the invitation with Roles "<roles>"
		// roles = app1pkg.NonExistentRole
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "app1pkg.NonExistentRole", true)
	})

	t.Run("invites: scn: Workspace owner cannot update an invitation with an invalid role set: duplicate role", func(t *testing.T) {
		// | roles                                                     |
		// | app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole       |
		// Given Workspace "Acme" has a pending invitation for "alice@example.com"
		// When Workspace Owner updates the invitation with Roles "<roles>"
		// roles = app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole
		// Then the response status is "400 Bad Request"
		testInvalidInvitesFeatureRoles(t, "app1pkg.LimitedAccessRole,app1pkg.LimitedAccessRole", true)
	})
}

func newInvitesFeatureFixture(t *testing.T, prefix string) *invitesFeatureFixture {
	t.Helper()
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	t.Cleanup(vit.TearDown)
	email := fmt.Sprintf("%s_%d@example.com", prefix, vit.NextNumber())
	login := vit.SignUp(email, "password", istructs.AppQName_test1_app1)
	principal := vit.SignIn(login)
	owner := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	ws := vit.CreateWorkspace(it.SimpleWSParams(prefix+"_"+vit.NextName()), owner)
	return &invitesFeatureFixture{
		vit:       vit,
		ws:        ws,
		login:     login,
		principal: principal,
		email:     email,
	}
}

func sendInvitesFeatureInvitation(t *testing.T, f *invitesFeatureFixture, email, roles string, expireDatetime int64) (istructs.RecordID, state.EmailMessage, string) {
	t.Helper()
	inviteID := InitiateInvitationByEMail(f.vit, f.ws, expireDatetime, email, roles, inviteEmailTemplate, inviteEmailSubject)
	message := f.vit.CaptureEmail()
	WaitForInviteState(f.vit, f.ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
	require.GreaterOrEqual(t, len(message.Body), 6)
	return inviteID, message, message.Body[:6]
}

func resendInvitesFeatureInvitation(t *testing.T, f *invitesFeatureFixture, inviteID istructs.RecordID, email, roles string) (state.EmailMessage, string) {
	t.Helper()
	newInviteID := InitiateInvitationByEMail(f.vit, f.ws, f.vit.Now().Add(time.Hour).UnixMilli(), email, roles, inviteEmailTemplate, inviteEmailSubject)
	require.Equal(t, istructs.NullRecordID, newInviteID)
	message := f.vit.CaptureEmail()
	WaitForInviteState(f.vit, f.ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
	require.GreaterOrEqual(t, len(message.Body), 6)
	return message, message.Body[:6]
}

func acceptInvitesFeatureInvitation(t *testing.T, f *invitesFeatureFixture, inviteID istructs.RecordID, principal *it.Principal, verificationCode string) {
	t.Helper()
	InitiateJoinWorkspace(f.vit, f.ws, inviteID, principal, verificationCode)
	WaitForInviteState(f.vit, f.ws, inviteID, invite.State_Invited, invite.State_Joined)
}

func setInvitesFeatureAlias(t *testing.T, f *invitesFeatureFixture) (*it.Principal, string) {
	t.Helper()
	alias := fmt.Sprintf("alias_%d@example.com", f.vit.NextNumber())
	systemToken := f.vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	initiateSetLoginAlias(t, f.vit, f.login, alias, systemToken)
	waitForLoginAlias(t, f.vit, f.login, alias)
	token := issuePrincipalToken(t, f.vit, f.login.Name, f.login.Pwd, f.login.AppQName)
	return &it.Principal{
		Login:       f.login,
		Token:       token,
		ProfileWSID: f.principal.ProfileWSID,
	}, alias
}

func getInvitesFeatureInvitation(t *testing.T, f *invitesFeatureFixture, inviteID istructs.RecordID) invitesFeatureInvite {
	t.Helper()
	row := f.vit.PostWS(f.ws, "q.sys.Collection", fmt.Sprintf(`
		{"args":{"Schema":"sys.Invite"},
		"elements":[{"fields":["Email","Roles","State","SubjectID","ActualLogin","InviteeProfileWSID","SubjectKind","sys.ID"]}],
		"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
	return invitesFeatureInvite{
		email:              row[0].(string),
		roles:              row[1].(string),
		state:              invite.State(row[2].(float64)),
		subjectID:          istructs.RecordID(row[3].(float64)),
		actualLogin:        row[4].(string),
		inviteeProfileWSID: istructs.WSID(row[5].(float64)),
		subjectKind:        istructs.SubjectKindType(row[6].(float64)),
	}
}

func getInvitesFeatureSubjects(t *testing.T, f *invitesFeatureFixture, login string) []invitesFeatureSubject {
	t.Helper()
	resp := f.vit.PostWS(f.ws, "q.sys.Collection", fmt.Sprintf(`
		{"args":{"Schema":"sys.Subject"},
		"elements":[{"fields":["sys.ID","Login","Roles","sys.IsActive"]}],
		"filters":[{"expr":"eq","args":{"field":"Login","value":"%s"}}]}`, login))
	if len(resp.Sections) == 0 {
		return nil
	}
	subjects := make([]invitesFeatureSubject, 0, len(resp.Sections[0].Elements))
	for i := range resp.Sections[0].Elements {
		row := resp.SectionRow(i)
		subjects = append(subjects, invitesFeatureSubject{
			id:       istructs.RecordID(row[0].(float64)),
			login:    row[1].(string),
			roles:    row[2].(string),
			isActive: row[3].(bool),
		})
	}
	return subjects
}

func requireInvitesFeatureMembership(t *testing.T, f *invitesFeatureFixture, login string, isActive bool) invitesFeatureSubject {
	t.Helper()
	subjects := getInvitesFeatureSubjects(t, f, login)
	require.Len(t, subjects, 1)
	require.Equal(t, isActive, subjects[0].isActive)
	return subjects[0]
}

func postInvitesFeatureInvitation(t *testing.T, f *invitesFeatureFixture, email, roles string, opts ...httpu.ReqOptFunc) {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"Email":"%s","Roles":"%s","ExpireDatetime":%d,"EmailTemplate":"%s","EmailSubject":"%s"}}`,
		email, roles, f.vit.Now().Add(time.Hour).UnixMilli(), inviteEmailTemplate, inviteEmailSubject)
	f.vit.PostWS(f.ws, "c.sys.InitiateInvitationByEMail", body, opts...)
}

func testInvalidInvitesFeatureRoles(t *testing.T, roles string, update bool) {
	t.Helper()
	f := newInvitesFeatureFixture(t, "invalid_roles")

	if !update {
		postInvitesFeatureInvitation(t, f, f.email, roles, httpu.Expect400())
		return
	}
	inviteID, _, _ := sendInvitesFeatureInvitation(t, f, f.email, initialRoles, f.vit.Now().Add(time.Hour).UnixMilli())
	body := fmt.Sprintf(`{"args":{"InviteID":%d,"Roles":"%s","EmailTemplate":"%s","EmailSubject":"%s"}}`,
		inviteID, roles, "text:"+invite.EmailTemplatePlaceholder_Roles, "roles updated")
	f.vit.PostWS(f.ws, "c.sys.InitiateUpdateInviteRoles", body, httpu.Expect400())
}

func invitationMessageFields(message state.EmailMessage) []string {
	return strings.Split(message.Body, ";")
}
