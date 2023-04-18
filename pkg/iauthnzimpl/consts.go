/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import "github.com/voedger/voedger/pkg/istructs"

var (
	qNameViewDeviceProfileWSIDIdx                   = istructs.NewQName(airPackage, "DeviceProfileWSIDIdx")
	qNameCDocWorkspaceKindRestaurant                = istructs.NewQName(airPackage, "Restaurant")
	qNameCDocWorkspaceKindAppWorkspace              = istructs.NewQName(istructs.SysPackage, "AppWorkspace")
	qNameCDocSubscriptionProfile                    = istructs.NewQName(airPackage, "SubscriptionProfile")
	qNameCDocUnTillOrders                           = istructs.NewQName(untillPackage, "orders")
	qNameCDocUnTillPBill                            = istructs.NewQName(untillPackage, "pbill")
	qNameTestDeniedCmd                              = istructs.NewQName(istructs.SysPackage, "TestDeniedCmd")
	qNameTestDeniedQry                              = istructs.NewQName(istructs.SysPackage, "TestDeniedQry")
	qNameTestDeniedCDoc                             = istructs.NewQName(istructs.SysPackage, "TestDeniedCDoc")
	qNameCDocLogin                                  = istructs.NewQName(istructs.SysPackage, "Login")
	qNameCDocChildWorkspace                         = istructs.NewQName(istructs.SysPackage, "ChildWorkspace")
	qNameCDocWorkspaceKindUser                      = istructs.NewQName(istructs.SysPackage, "UserProfile")
	qNameCDocWorkspaceKindDevice                    = istructs.NewQName(istructs.SysPackage, "DeviceProfile")
	qNameCDocWorkspaceDescriptor                    = istructs.NewQName(istructs.SysPackage, "WorkspaceDescriptor")
	qNameCmdUpdateSubscription                      = istructs.NewQName(airPackage, "UpdateSubscription")
	qNameCmdStoreSubscriptionProfile                = istructs.NewQName(airPackage, "StoreSubscriptionProfile")
	qNameCmdLinkDeviceToRestaurant                  = istructs.NewQName(airPackage, "LinkDeviceToRestaurant")
	qNameQryIssuePrincipalToken                     = istructs.NewQName(istructs.SysPackage, "IssuePrincipalToken")
	qNameCmdCreateLogin                             = istructs.NewQName(istructs.SysPackage, "CreateLogin")
	qNameQryEcho                                    = istructs.NewQName(istructs.SysPackage, "Echo")
	qNameQryGRCount                                 = istructs.NewQName(istructs.SysPackage, "GRCount")
	qNameCmdSendEmailVerificationCode               = istructs.NewQName(istructs.SysPackage, "SendEmailVerificationCode")
	qNameCmdResetPasswordByEmail                    = istructs.NewQName(istructs.SysPackage, "ResetPasswordByEmail")
	qNameQryInitiateResetPasswordByEmail            = istructs.NewQName(istructs.SysPackage, "InitiateResetPasswordByEmail")
	qNameQryIssueVerifiedValueTokenForResetPassword = istructs.NewQName(istructs.SysPackage, "IssueVerifiedValueTokenForResetPassword")
	qNameQryDescribePackageNames                    = istructs.NewQName(istructs.SysPackage, "DescribePackageNames")
	qNameQryDescribePackage                         = istructs.NewQName(istructs.SysPackage, "DescribePackage")
	qNameCmdInitiateJoinWorkspace                   = istructs.NewQName(istructs.SysPackage, "InitiateJoinWorkspace")
	qNameCmdInitiateLeaveWorkspace                  = istructs.NewQName(istructs.SysPackage, "InitiateLeaveWorkspace")
	qNameCmdChangePassword                          = istructs.NewQName(istructs.SysPackage, "ChangePassword")
	qNameCmdInitiateInvitationByEmail               = istructs.NewQName(istructs.SysPackage, "InitiateInvitationByEMail")
	qNameQryCollection                              = istructs.NewQName(istructs.SysPackage, "Collection")
	qNameCmdInitiateUpdateInviteRoles               = istructs.NewQName(istructs.SysPackage, "InitiateUpdateInviteRoles")
	qNameCmdInitiateCancelAcceptedInvite            = istructs.NewQName(istructs.SysPackage, "InitiateCancelAcceptedInvite")
	qNameCmdCancelSendInvite                        = istructs.NewQName(istructs.SysPackage, "CancelSendInvite")
	qNameCmdInitChildWorkspace                      = istructs.NewQName(istructs.SysPackage, "InitChildWorkspace")
	qNameCmdEnrichPrincipalToken                    = istructs.NewQName(istructs.SysPackage, "EnrichPrincipalToken")
	qNameCDocUPProfile                              = istructs.NewQName(airPackage, "UPProfile")
	qNameCDocResellerSubscriptionsProfile           = istructs.NewQName(airPackage, "ResellerSubscriptionsProfile")
	qNameCmdCreateUPProfile                         = istructs.NewQName(airPackage, "CreateUPProfile")
	qNameQryGetUPOnboardingPage                     = istructs.NewQName(airPackage, "GetUPOnboardingPage")
	qNameQryGetUPVerificationStatus                 = istructs.NewQName(airPackage, "GetUPVerificationStatus")
	qNameQryGetUPAccountStatus                      = istructs.NewQName(airPackage, "GetUPAccountStatus")
	qNameQryGetUPEventHistory                       = istructs.NewQName(airPackage, "GetUPEventHistory")
	qNameCmdStoreResellerSubscriptionsProfile       = istructs.NewQName(airPackage, "StoreResellerSubscriptionsProfile")
	qNameQryGetHostedAirSubscriptions               = istructs.NewQName(airPackage, "GetHostedAirSubscriptions")
	qNameQryGetUPStatus                             = istructs.NewQName(airPackage, "GetUPStatus")
	qNameQryQueryResellerInfo                       = istructs.NewQName(airPackage, "QueryResellerInfo")
	qNameCmdCreateUntillPayment                     = istructs.NewQName(airPackage, "CreateUntillPayment")

	// Air roles
	qNameRoleResellersAdmin         = istructs.NewQName(airPackage, "ResellersAdmin")
	qNameRoleUntillPaymentsReseller = istructs.NewQName(airPackage, "UntillPaymentsReseller")
	qNameRoleUntillPaymentsUser     = istructs.NewQName(airPackage, "UntillPaymentsUser")
	qNameRoleAirReseller            = istructs.NewQName(airPackage, "AirReseller")
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
