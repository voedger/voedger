/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package oldacl

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
)

func IsOperationAllowed(opKind appdef.OperationKind, qName appdef.QName, fields []string, principals []iauthnz.Principal) bool {
	policy := appdef.PolicyKind_Deny
	var lastDenyingACElem ACElem
	for _, acElem := range defaultACL {
		if matchOrNotSpecified_OpKinds(acElem.pattern.opKindsPattern, opKind) &&
			matchOrNotSpecified_QNames(acElem.pattern.qNamesPattern, qName) &&
			matchOrNotSpecified_Fields(acElem.pattern.fieldsPattern, fields) &&
			matchOrNotSpecified_Principals(acElem.pattern.principalsPattern, principals) {
			if policy = acElem.policy; policy == appdef.PolicyKind_Deny {
				lastDenyingACElem = acElem
			}
		}
	}
	if policy == appdef.PolicyKind_Deny && logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("old-style %s %s for %s: %s -> deny", opKind, qName, prnsToString(principals), lastDenyingACElem.desc))
	}
	return policy == appdef.PolicyKind_Allow
}

var defaultACL = ACL{
	{
		desc: "null auth policy",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdLinkDeviceToRestaurant,
				// https://dev.untill.com/projects/#!688808
				qNameQryGetDigitalReceipt,
				// https://dev.untill.com/projects/#!688808
				qNameQrySendReceiptByEmail,
				// https://dev.untill.com/projects/#!698913
				qNameQryQueryResellerInfo,
				// https://dev.untill.com/projects/#!700365
				qNameQryGetResellers,
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "allowed to sys.Guest login, i.e. without principal token at all",
		pattern: PatternType{
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_User, WSID: istructs.GuestWSID}}},
			qNamesPattern: []appdef.QName{
				qNameCmdProvideCertificatePart, qNameCmdProvideCertificate, qNameQryGetCustomerStatus,
				qNameCmdFiscalizeDocument, qNameQryFiscalizationResultStatus, qNameCmdCreateExport, qNameQryExportStatus, qNameQryValidateDocument},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "everything is allowed to WorkspaceOwner",
		pattern: PatternType{
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceOwner}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "deny all on few QNames from all",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdStoreSubscriptionProfile, qNameCmdUpdateSubscription,

				qNameCDocUnTillOrders, qNameCDocUnTillPBill,
				qNameTestDeniedCmd, qNameTestDeniedCDoc, qNameCDocLogin, qNameCDocChildWorkspace, qNameTestDeniedQry, qNameTestDeniedCmd_it, qNameTestDeniedQry_it,
			},
		},
		policy: appdef.PolicyKind_Deny,
	},
	{
		desc: "grant select only on few documents to WorkspaceOwner",
		pattern: PatternType{
			opKindsPattern:    []appdef.OperationKind{appdef.OperationKind_Select},
			qNamesPattern:     []appdef.QName{qNameCDocChildWorkspace},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceOwner}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "revoke insert or update on wdoc.air.LastNumbers from all",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update},
			qNamesPattern:  []appdef.QName{qNameWDocLastNumbers},
		},
		policy: appdef.PolicyKind_Deny,
	},
	{
		// DENY ALL FROM LOGIN 'untillchargebeeagent'
		desc: "deny all from 'untillchargebeeagent' login",
		pattern: PatternType{
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_User, Name: untillChargebeeAgentLogin}}},
		},
		policy: appdef.PolicyKind_Deny,
	},
	{
		// GRANT EXEC ON c.air.UpdateSubscription TO ROLE ProfileUser AND LOGIN 'untillchargebeeagent'
		desc: "c.air.UpdateSubscription is allowed for 'untillchargebeeagent' login only and in its profile only",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{qNameCmdUpdateSubscription},
			principalsPattern: [][]iauthnz.Principal{
				{
					{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleProfileOwner},
					// AND
					{Kind: iauthnz.PrincipalKind_User, Name: untillChargebeeAgentLogin},
				},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// WorkspaceAdmin role asssigned automatically if has e.g. RoleResellersAdmin or RoleUntillPaymentsReseller
		desc: "allow few reseller-related commands to WorkspaceAdmin",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdInitiateInvitationByEmail,
				qNameQryCollection,
				qNameCmdInitiateUpdateInviteRoles,
				qNameCmdInitiateCancelAcceptedInvite,
				qNameCmdCancelSentInvite,
				qNameCDocChildWorkspace,
				qNameCmdInitChildWorkspace,
				qNameCmdEnrichPrincipalToken,
				istructs.QNameCommandCUD,

				// https://github.com/voedger/voedger/issues/208
				qNameCmdInitiateDeactivateWorkspace,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceAdmin}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// https://github.com/voedger/voedger/issues/125
		desc: "grant UPDATE on air.UntillPayments to role sys.WorkspaceAdmin",
		pattern: PatternType{
			qNamesPattern:     []appdef.QName{qNameCDocUntillPayments},
			opKindsPattern:    []appdef.OperationKind{appdef.OperationKind_Update},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceAdmin}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow SELECT cdoc.air.ResellerSubscriptionsProfile to air.SubscriptionReseller",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Select},
			qNamesPattern:  []appdef.QName{qNameCDocResellerSubscriptionsProfile},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleAirReseller}}, // deprecated
				// OR
				// https://dev.untill.com/projects/#!694587
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleSubscriptionReseller}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow exec few portals-related funcs to air.SubscriptionReseller",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdStoreResellerSubscriptionsProfile,
				qNameQryGetHostedAirSubscriptions,
				qNameQryCollection,

				// https://dev.untill.com/projects/#!638320
				qNameQryGetUPStatus,
				// https://dev.untill.com/projects/#!673032
				qNameQryListIssuedSubscriptionInvoices,
				qNameQryListIssuedSubscriptionResellers,
				qNameQryListPaidSubscriptionInvoices,
				qNameQryListPaidSubscriptionResellers,
				// https://dev.untill.com/projects/#!679811
				qNameQryIsDirectReseller,
				// https://dev.untill.com/projects/#!675263
				qNameQryPaidSubscriptionInvoicesReport,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleAirReseller}}, // deprecated
				// OR
				// https://dev.untill.com/projects/#!694587
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleSubscriptionReseller}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow SELECT cdoc.air.UPProfile to air.UntillPaymentsReseller and air.UntillPaymentsUser",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Select},
			qNamesPattern:  []appdef.QName{qNameCDocUPProfile},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// ACL for portals https://dev.untill.com/projects/#!637208
		desc: "allow few portal-related funcs to air.UntillPaymentsReseller and air.UntillPaymentsUser",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdCreateUPProfile,
				qNameQryGetUPOnboardingPage,
				qNameQryGetUPVerificationStatus,
				qNameQryGetUPAccountStatus,
				qNameQryGetUPEventHistory,
				qNameQryCollection,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// ACL for FiscalCloud
		desc: "allow FiscalCloud onboarding functions to role fiscalcloud.OnboardSite",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdAddCustomer,
				qNameCmdUpdateCustomer,
				qNameCmdDeactivateCustomer,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleFiscalCloudOnboardSite}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "grant exec on few funcs to role air.UntillPaymentsUser",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameQryGetUPStatus,
				qNameCmdCreateUntillPayment,

				// https://github.com/voedger/voedger/issues/57
				qNameCmdEnsureUPPredefinedPaymentModesExist,

				// https://dev.untill.com/projects/#!641315
				qNameQryGetUPTerminals,
				qNameQryActivateUPTerminal,
				qNameQryGetUPPaymentMethods,
				qNameQryToggleUPPaymentMethod,
				qNameQryRequestUPPaymentMethod,
				qNameQryGetUPTransactionsOverview,
				qNameQryGetUPTransactionReceipts,
				// https://dev.untill.com/projects/#!664899
				qNameQryGetUPLocationSubjects,
				// https://dev.untill.com/projects/#!659825
				qNameQryGetLocationDailyUPReport,
				// https://dev.untill.com/projects/#!653069
				qNameCmdVoidUntillPayment,
				// https://dev.untill.com/projects/#!683625
				qNameQryCreateTap2PaySession,
				// https://dev.untill.com/projects/#!693712
				qNameCmdSaveTap2PayPayment,
				// https://untill.atlassian.net/browse/AIR-47
				qNameQryShowBillOnDisplay,
				qNameQryShowOrderOnDisplay,
				qNameQryShowStandbyOnDisplay,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// https://dev.untill.com/projects/#!640535
		desc: "grant exec on c.air.RegenerateUPProfileApiToken to role air.UntillPaymentsReseller and air.UntillPaymentsUser",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Execute},
			qNamesPattern:  []appdef.QName{qNameCmdRegenerateUPProfileApiToken},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "grant exec on q.air.UPTerminalWebhook to role air.UntillPaymentsTerminal",
		pattern: PatternType{
			qNamesPattern:     []appdef.QName{qNameQryUPTerminalWebhook},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsTerminal}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// https://github.com/voedger/voedger/issues/422
		// https://dev.untill.com/projects/#!649352
		// https://dev.untill.com/projects/#!650998
		// https://dev.untill.com/projects/#!653137
		// https://dev.untill.com/projects/#!665805
		// https://dev.untill.com/projects/#!663035
		desc: "grant exec on few funcs to role air.UntillPaymentsReseller and role air.UntillPaymentsUser",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameQryGetUPPayouts,
				qNameQryGetUPInvoiceParties,
				qNameQryGetUPTransferInstrument,
				qNameCmdRetryTransferUPPayout,
				// https://dev.untill.com/projects/#!685617
				qNameQryGetUPLocationRates,
				// https://dev.untill.com/projects/#!685179
				qNameQryUpdateShopperStatement,
				// https://dev.untill.com/projects/#!710217
				qNameQryGetUPPayoutTransfers,
				qNameQryGetUPInvoices,
				qNameCmdUpdateUPProfile,
			},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "grant exec on few funcs to role air.UntillPaymentsReseller",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameCmdUpdateUPLocationRates,
				qNameQryGetUPFeesOverview,
				// https://dev.untill.com/projects/#!664876
				qNameQryIsDirectReseller,
				// https://dev.untill.com/projects/#!659825
				qNameQryGetResellerDailyUPReport,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "grant exec on few funcs to role air.UntillPaymentsManager",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameQryGetAllUPPayouts,
				qNameQryGetUPLocationInvoiceParties,
				// https://dev.untill.com/projects/#!710217
				qNameQryGetAllUPInvoices,
				qNameQryGetAllUPPayoutTransfers,
				// https://dev.untill.com/projects/#!711418
				qNameQryGetDailyUPReports,
				// https://dev.untill.com/projects/#!710982
				qNameQryGetUPVATTransfers,
				qNameQryGetUPBeneficiaryVATDebts,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsManager}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "allow update cdoc.air.Reseller to sys.RoleWorkspaceAdmin",
		pattern: PatternType{
			opKindsPattern:    []appdef.OperationKind{appdef.OperationKind_Update},
			qNamesPattern:     []appdef.QName{qNameCDocReseller},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceAdmin}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// https://github.com/voedger/voedger/issues/2470
		// https://github.com/voedger/voedger/issues/3007
		desc: "grant exec on q.sys.State, sys.RegisterTempBLOB1d, q.sys.DownloadBLOBAuthnz to role.air.BOReader",
		pattern: PatternType{
			opKindsPattern:    []appdef.OperationKind{appdef.OperationKind_Execute},
			qNamesPattern:     []appdef.QName{qNameQryState, qNameCmdRegisterTempBLOB1d, qNameQryDownloadBLOBAuthnz},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleBOReader}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// https://github.com/voedger/voedger/issues/2470
		desc: "grant select on field q.sys.State.State to role.air.BOReader",
		pattern: PatternType{
			opKindsPattern:    []appdef.OperationKind{appdef.OperationKind_Select},
			fieldsPattern:     [][]string{{"State"}},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleBOReader}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "grant exec on few funcs to role air.ResellerPortalDashboardViewer",
		pattern: PatternType{
			qNamesPattern: []appdef.QName{
				qNameQryGetAirLocations,
				qNameQryResellersDashboardSalesMetrics,
				qNameQryResellersDashboardBackofficeMetrics,
			},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameResellerPortalDashboardViewer}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "grant select on table air.UntillPayments to air.UntillPaymentsUser",
		pattern: PatternType{
			opKindsPattern:    []appdef.OperationKind{appdef.OperationKind_Select},
			qNamesPattern:     []appdef.QName{qNameCDocUntillPayments},
			principalsPattern: [][]iauthnz.Principal{{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsUser}}},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		// TODO: carefully check which docs are able to be read by whom
		desc: "grant select on few tables to air.AirReseller and air.UntillPaymentsReseller and SubscriptionReseller and WorkspaceAdmin",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Select},
			qNamesPattern:  []appdef.QName{qNameCDocReseller, qNameCDocUntillPayments},
			principalsPattern: [][]iauthnz.Principal{
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleAirReseller}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleUntillPaymentsReseller}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: qNameRoleSubscriptionReseller}},
				// OR
				{{Kind: iauthnz.PrincipalKind_Role, QName: iauthnz.QNameRoleWorkspaceAdmin}},
			},
		},
		policy: appdef.PolicyKind_Allow,
	},
	{
		desc: "do not grant exec on c.air.TestSubscriptionProfile to anyone",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Execute},
			qNamesPattern: []appdef.QName{appdef.NewQName(airPackage, "TestSubscriptionProfile")},
		},
		policy: appdef.PolicyKind_Deny,
	},
	{
		desc: "do not grant insert and update on cdoc.air.SubscriptionProfile to anyone",
		pattern: PatternType{
			opKindsPattern: []appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update},
			qNamesPattern: []appdef.QName{appdef.NewQName(airPackage, "SubscriptionProfile")},
		},
		policy: appdef.PolicyKind_Deny,
	},

	// SubscriptionProfile
}
