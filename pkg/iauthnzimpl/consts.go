/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"github.com/voedger/voedger/pkg/schemas"
)

var (
	qNameViewDeviceProfileWSIDIdx                   = schemas.NewQName(airPackage, "DeviceProfileWSIDIdx")
	qNameCDocWorkspaceKindRestaurant                = schemas.NewQName(airPackage, "Restaurant")
	qNameCDocWorkspaceKindAppWorkspace              = schemas.NewQName(schemas.SysPackage, "AppWorkspace")
	qNameCDocSubscriptionProfile                    = schemas.NewQName(airPackage, "SubscriptionProfile")
	qNameCDocUnTillOrders                           = schemas.NewQName(untillPackage, "orders")
	qNameCDocUnTillPBill                            = schemas.NewQName(untillPackage, "pbill")
	qNameTestDeniedCmd                              = schemas.NewQName(schemas.SysPackage, "TestDeniedCmd")
	qNameTestDeniedQry                              = schemas.NewQName(schemas.SysPackage, "TestDeniedQry")
	qNameTestDeniedCDoc                             = schemas.NewQName(schemas.SysPackage, "TestDeniedCDoc")
	qNameCDocLogin                                  = schemas.NewQName(schemas.SysPackage, "Login")
	qNameCDocChildWorkspace                         = schemas.NewQName(schemas.SysPackage, "ChildWorkspace")
	qNameCDocWorkspaceKindUser                      = schemas.NewQName(schemas.SysPackage, "UserProfile")
	qNameCDocWorkspaceKindDevice                    = schemas.NewQName(schemas.SysPackage, "DeviceProfile")
	qNameCDocWorkspaceDescriptor                    = schemas.NewQName(schemas.SysPackage, "WorkspaceDescriptor")
	qNameCmdUpdateSubscription                      = schemas.NewQName(airPackage, "UpdateSubscription")
	qNameCmdStoreSubscriptionProfile                = schemas.NewQName(airPackage, "StoreSubscriptionProfile")
	qNameCmdLinkDeviceToRestaurant                  = schemas.NewQName(airPackage, "LinkDeviceToRestaurant")
	qNameQryIssuePrincipalToken                     = schemas.NewQName(schemas.SysPackage, "IssuePrincipalToken")
	qNameCmdCreateLogin                             = schemas.NewQName(schemas.SysPackage, "CreateLogin")
	qNameQryEcho                                    = schemas.NewQName(schemas.SysPackage, "Echo")
	qNameQryGRCount                                 = schemas.NewQName(schemas.SysPackage, "GRCount")
	qNameCmdSendEmailVerificationCode               = schemas.NewQName(schemas.SysPackage, "SendEmailVerificationCode")
	qNameCmdResetPasswordByEmail                    = schemas.NewQName(schemas.SysPackage, "ResetPasswordByEmail")
	qNameQryInitiateResetPasswordByEmail            = schemas.NewQName(schemas.SysPackage, "InitiateResetPasswordByEmail")
	qNameQryIssueVerifiedValueTokenForResetPassword = schemas.NewQName(schemas.SysPackage, "IssueVerifiedValueTokenForResetPassword")
	qNameQryDescribePackageNames                    = schemas.NewQName(schemas.SysPackage, "DescribePackageNames")
	qNameQryDescribePackage                         = schemas.NewQName(schemas.SysPackage, "DescribePackage")
	qNameCmdInitiateJoinWorkspace                   = schemas.NewQName(schemas.SysPackage, "InitiateJoinWorkspace")
	qNameCmdInitiateLeaveWorkspace                  = schemas.NewQName(schemas.SysPackage, "InitiateLeaveWorkspace")
	qNameCmdChangePassword                          = schemas.NewQName(schemas.SysPackage, "ChangePassword")
	qNameCmdInitiateInvitationByEmail               = schemas.NewQName(schemas.SysPackage, "InitiateInvitationByEMail")
	qNameQryCollection                              = schemas.NewQName(schemas.SysPackage, "Collection")
	qNameCmdInitiateUpdateInviteRoles               = schemas.NewQName(schemas.SysPackage, "InitiateUpdateInviteRoles")
	qNameCmdInitiateCancelAcceptedInvite            = schemas.NewQName(schemas.SysPackage, "InitiateCancelAcceptedInvite")
	qNameCmdCancelSendInvite                        = schemas.NewQName(schemas.SysPackage, "CancelSendInvite")
	qNameCmdInitChildWorkspace                      = schemas.NewQName(schemas.SysPackage, "InitChildWorkspace")
	qNameCmdEnrichPrincipalToken                    = schemas.NewQName(schemas.SysPackage, "EnrichPrincipalToken")
	qNameCDocUPProfile                              = schemas.NewQName(airPackage, "UPProfile")
	qNameCDocResellerSubscriptionsProfile           = schemas.NewQName(airPackage, "ResellerSubscriptionsProfile")
	qNameCmdCreateUPProfile                         = schemas.NewQName(airPackage, "CreateUPProfile")
	qNameQryGetUPOnboardingPage                     = schemas.NewQName(airPackage, "GetUPOnboardingPage")
	qNameQryGetUPVerificationStatus                 = schemas.NewQName(airPackage, "GetUPVerificationStatus")
	qNameQryGetUPAccountStatus                      = schemas.NewQName(airPackage, "GetUPAccountStatus")
	qNameQryGetUPEventHistory                       = schemas.NewQName(airPackage, "GetUPEventHistory")
	qNameCmdStoreResellerSubscriptionsProfile       = schemas.NewQName(airPackage, "StoreResellerSubscriptionsProfile")
	qNameQryGetHostedAirSubscriptions               = schemas.NewQName(airPackage, "GetHostedAirSubscriptions")
	qNameQryGetUPStatus                             = schemas.NewQName(airPackage, "GetUPStatus")
	qNameQryQueryResellerInfo                       = schemas.NewQName(airPackage, "QueryResellerInfo")
	qNameCmdCreateUntillPayment                     = schemas.NewQName(airPackage, "CreateUntillPayment")
	qNameCmdRegenerateUPProfileApiToken             = schemas.NewQName(airPackage, "RegenerateUPProfileApiToken")
	qNameCmdEnsureUPPredefinedPaymentModesExist     = schemas.NewQName(airPackage, "EnsureUPPredefinedPaymentModesExist")
	qNameQryGetUPTerminals                          = schemas.NewQName(airPackage, "GetUPTerminals")
	qNameQryActivateUPTerminal                      = schemas.NewQName(airPackage, "ActivateUPTerminal")
	qNameQryGetUPPaymentMethods                     = schemas.NewQName(airPackage, "GetUPPaymentMethods")
	qNameQryToggleUPPaymentMethod                   = schemas.NewQName(airPackage, "ToggleUPPaymentMethod")
	qNameQryRequestUPPaymentMethod                  = schemas.NewQName(airPackage, "RequestUPPaymentMethod")
	qNameQryUPTerminalWebhook                       = schemas.NewQName(airPackage, "UPTerminalWebhook")

	// Air roles
	qNameRoleResellersAdmin         = schemas.NewQName(airPackage, "ResellersAdmin")
	qNameRoleUntillPaymentsReseller = schemas.NewQName(airPackage, "UntillPaymentsReseller")
	qNameRoleUntillPaymentsUser     = schemas.NewQName(airPackage, "UntillPaymentsUser")
	qNameRoleAirReseller            = schemas.NewQName(airPackage, "AirReseller")
	qNameRoleUntillPaymentsTerminal = schemas.NewQName(airPackage, "UntillPaymentsTerminal")
)

const (
	field_DeviceProfileWSID     = "DeviceProfileWSID"
	field_ComputersID           = "ComputersID"
	field_RestaurantComputersID = "RestaurantComputersID"
	field_dummy                 = "dummy"
	field_OwnerWSID             = "OwnerWSID"
	airPackage                  = "air"
	untillPackage               = "untill"
	untillChargebeeAgentLogin   = "untillchargebeeagent"
)

const (
	ACPolicy_Deny ACPolicyType = iota
	ACPolicy_Allow
)
