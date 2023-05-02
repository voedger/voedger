/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/voedger/pkg/iauthnz"
	"github.com/untillpro/voedger/pkg/istructs"
)

func (acl ACL) IsAllowed(principals []iauthnz.Principal, req iauthnz.AuthzRequest) bool {
	policy := ACPolicy_Deny
	var lastDenyingACElem ACElem
	for _, acElem := range acl {
		if matchOrNotSpecified_OpKinds(acElem.pattern.opKindsPattern, req.OperationKind) &&
			matchOrNotSpecified_QNames(acElem.pattern.qNamesPattern, req.Resource) &&
			matchOrNotSpecified_Principals(acElem.pattern.principalsPattern, principals) {
			if policy = acElem.policy; policy == ACPolicy_Deny {
				lastDenyingACElem = acElem
			}
		}
	}
	if policy == ACPolicy_Deny && logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("%s for %s: %s -> deny", authNZToString(req), prnsToString(principals), lastDenyingACElem.desc))
	}
	return policy == ACPolicy_Allow
}

var defaultACL = ACL{
	{
		desc: "null auth policy",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdLinkDeviceToRestaurant,
				qNameQryIssuePrincipalToken,
				qNameCmdCreateLogin,
				qNameQryEcho,
				qNameQryGRCount,
				qNameCmdResetPasswordByEmail,
				qNameQryInitiateResetPasswordByEmail,
				qNameQryIssueVerifiedValueTokenForResetPassword,
				qNameCmdChangePassword,
			},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "everything is allowed to WorkspaceSubject",
		pattern: PatternType{
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceSubject}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "deny all on few QNames from all",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdStoreSubscriptionProfile, qNameCmdUpdateSubscription,

				qNameCDocSubscriptionProfile, qNameCDocUnTillOrders, qNameCDocUnTillPBill,
				qNameTestDeniedCmd, qNameTestDeniedCDoc, qNameCDocLogin, qNameCDocChildWorkspace, qNameTestDeniedQry,

				qNameCDocWorkspaceKindUser,
				qNameCDocWorkspaceKindDevice,
				qNameCDocWorkspaceKindRestaurant,
				qNameCDocWorkspaceKindAppWorkspace,
				qNameCmdSendEmailVerificationCode,

				qNameQryDescribePackage,
				qNameQryDescribePackageNames,
			},
		},
		policy: ACPolicy_Deny,
	},
	{
		desc: "update only is allowed for CDoc<$wsKind> for WorkspaceSubject",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCDocWorkspaceKindUser,
				qNameCDocWorkspaceKindDevice,
				qNameCDocWorkspaceKindRestaurant,
				qNameCDocWorkspaceKindAppWorkspace,
			},
			opKindsPattern:    []iauthnz.OperationKindType{iauthnz.OperationKind_UPDATE},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceSubject}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		// DENY ALL FROM LOGIN 'untillchargebeeagent'
		desc: "deny all from 'untillchargebeeagent' login",
		pattern: PatternType{
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_User, Name: untillChargebeeAgentLogin}}},
		},
		policy: ACPolicy_Deny,
	},
	{
		// GRANT EXEC ON c.air.UpdateSubscription TO ROLE ProfileUser AND LOGIN 'untillchargebeeagent'
		desc: "c.air.UpdateSubscription is allowed for 'untillchargebeeagent' login only and in its profile only",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{qNameCmdUpdateSubscription},
			principalsPattern: [][]iauthnz.Principal{
				{
					// AND
					{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleProfileOwner},
					{Kind: iauthnz.PrincipalKind_User, Name: untillChargebeeAgentLogin},
				},
			},
		},
		policy: ACPolicy_Allow,
	},
	{
		// GRANT SELECT q.sys.DescribePackage* TO ROLE ProfileUser
		desc: "q.sys.DescribePackage* is allowed to be called in profile only",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameQryDescribePackage,
				qNameQryDescribePackageNames,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleProfileOwner}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "c.sys.InitiateJoinWorkspace is allowed for authenticated users",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdInitiateJoinWorkspace,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_User}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "c.sys.InitiateLeaveWorkspace is allowed for authenticated users",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdInitiateLeaveWorkspace,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_User}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		// WorkspaceAdmin role asssigned automatically if has e.g. RoleResellersAdmin or RoleUntillPaymentsReseller
		desc: "allow few reseller-related commands to WorkspaceAdmin",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdInitiateInvitationByEmail,
				qNameQryCollection,
				qNameCmdInitiateUpdateInviteRoles,
				qNameCmdInitiateCancelAcceptedInvite,
				qNameCmdCancelSendInvite,
				qNameCDocChildWorkspace,
				qNameCmdInitChildWorkspace,
				qNameCmdEnrichPrincipalToken,
				istructs.QNameCommandCUD,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceAdmin}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow SELECT cdoc.air.ResellerSubscriptionsProfile to air.AirReseller",
		pattern: PatternType{
			opKindsPattern:    []iauthnz.OperationKindType{iauthnz.OperationKind_SELECT},
			qNamesPattern:     []istructs.QName{qNameCDocResellerSubscriptionsProfile},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleAirReseller}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow exec few portals-related funcs to air.AirReseller",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdStoreResellerSubscriptionsProfile,
				qNameQryGetHostedAirSubscriptions,
				qNameQryCollection,

				// https://dev.untill.com/projects/#!638320
				qNameQryGetUPStatus,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleAirReseller}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow SELECT cdoc.air.UPProfile to air.UntillPaymentsReseller and air.AirReseller",
		pattern: PatternType{
			opKindsPattern: []iauthnz.OperationKindType{iauthnz.OperationKind_SELECT},
			qNamesPattern:  []istructs.QName{qNameCDocUPProfile},
			principalsPattern: [][]iauthnz.Principal{
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
			},
		},
		policy: ACPolicy_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow few portal-related funcs to air.UntillPaymentsReseller and air.UntillPaymentsUser",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameCmdCreateUPProfile,
				qNameQryGetUPOnboardingPage,
				qNameQryGetUPVerificationStatus,
				qNameQryGetUPAccountStatus,
				qNameQryGetUPEventHistory,
				qNameQryCollection,
			},
			principalsPattern: [][]iauthnz.Principal{
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
			},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "q.air.QueryResellerInfo is allowed for authenticated users",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameQryQueryResellerInfo,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_User}}},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "grant exec on c.air.CreateUntillPayment, q.air.GetUPStatus to role air.UntillPaymentsUser",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameQryGetUPStatus,
				qNameCmdCreateUntillPayment,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
			},
		},
		policy: ACPolicy_Allow,
	},
	{
		desc: "grant exec on q.air.UPTerminalWebhook to role air.UntillPaymentsTerminal",
		pattern: PatternType{
			qNamesPattern: []istructs.QName{
				qNameQryUPTerminalWebhook,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsTerminal}},
			},
		},
		policy: ACPolicy_Allow,
	},
}
