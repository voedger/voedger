/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package oldacl

import (
	"github.com/voedger/voedger/pkg/appdef"
)

var (
	qNameCDocUnTillOrders                       = appdef.NewQName(untillPackage, "orders")
	qNameCDocUnTillPBill                        = appdef.NewQName(untillPackage, "pbill")
	qNameTestDeniedCmd                          = appdef.NewQName(appdef.SysPackage, "TestDeniedCmd")
	qNameTestDeniedCmd_it                       = appdef.NewQName("app1pkg", "TestDeniedCmd")
	qNameTestDeniedQry_it                       = appdef.NewQName("app1pkg", "TestDeniedQuery")
	qNameTestDeniedQry                          = appdef.NewQName(appdef.SysPackage, "TestDeniedQry")
	qNameTestDeniedCDoc                         = appdef.NewQName("app1pkg", "TestDeniedCDoc")
	qNameCDocLogin                              = appdef.NewQName(registryPackage, "Login")
	qNameCDocChildWorkspace                     = appdef.NewQName(appdef.SysPackage, "ChildWorkspace")
	qNameCmdUpdateSubscription                  = appdef.NewQName(airPackage, "UpdateSubscription")
	qNameCmdStoreSubscriptionProfile            = appdef.NewQName(airPackage, "StoreSubscriptionProfile")
	qNameCmdLinkDeviceToRestaurant              = appdef.NewQName(airPackage, "LinkDeviceToRestaurant")
	qNameCmdInitiateInvitationByEmail           = appdef.NewQName(appdef.SysPackage, "InitiateInvitationByEMail")
	qNameQryCollection                          = appdef.NewQName(appdef.SysPackage, "Collection")
	qNameCmdInitiateUpdateInviteRoles           = appdef.NewQName(appdef.SysPackage, "InitiateUpdateInviteRoles")
	qNameCmdInitiateCancelAcceptedInvite        = appdef.NewQName(appdef.SysPackage, "InitiateCancelAcceptedInvite")
	qNameCmdCancelSentInvite                    = appdef.NewQName(appdef.SysPackage, "CancelSentInvite")
	qNameCmdInitChildWorkspace                  = appdef.NewQName(appdef.SysPackage, "InitChildWorkspace")
	qNameCmdEnrichPrincipalToken                = appdef.NewQName(appdef.SysPackage, "EnrichPrincipalToken")
	qNameCDocUPProfile                          = appdef.NewQName(airPackage, "UPProfile")
	qNameCDocResellerSubscriptionsProfile       = appdef.NewQName(airPackage, "ResellerSubscriptionsProfile")
	qNameCmdCreateUPProfile                     = appdef.NewQName(airPackage, "CreateUPProfile")
	qNameQryGetUPOnboardingPage                 = appdef.NewQName(airPackage, "GetUPOnboardingPage")
	qNameQryGetUPVerificationStatus             = appdef.NewQName(airPackage, "GetUPVerificationStatus")
	qNameQryGetUPAccountStatus                  = appdef.NewQName(airPackage, "GetUPAccountStatus")
	qNameQryGetUPEventHistory                   = appdef.NewQName(airPackage, "GetUPEventHistory")
	qNameCmdStoreResellerSubscriptionsProfile   = appdef.NewQName(airPackage, "StoreResellerSubscriptionsProfile")
	qNameQryGetHostedAirSubscriptions           = appdef.NewQName(airPackage, "GetHostedAirSubscriptions")
	qNameQryGetUPStatus                         = appdef.NewQName(airPackage, "GetUPStatus")
	qNameQryQueryResellerInfo                   = appdef.NewQName(airPackage, "QueryResellerInfo")
	qNameCmdCreateUntillPayment                 = appdef.NewQName(airPackage, "CreateUntillPayment")
	qNameCmdRegenerateUPProfileApiToken         = appdef.NewQName(airPackage, "RegenerateUPProfileApiToken")
	qNameCmdEnsureUPPredefinedPaymentModesExist = appdef.NewQName(airPackage, "EnsureUPPredefinedPaymentModesExist")
	qNameQryGetUPTerminals                      = appdef.NewQName(airPackage, "GetUPTerminals")
	qNameQryActivateUPTerminal                  = appdef.NewQName(airPackage, "ActivateUPTerminal")
	qNameQryGetUPPaymentMethods                 = appdef.NewQName(airPackage, "GetUPPaymentMethods")
	qNameQryToggleUPPaymentMethod               = appdef.NewQName(airPackage, "ToggleUPPaymentMethod")
	qNameQryRequestUPPaymentMethod              = appdef.NewQName(airPackage, "RequestUPPaymentMethod")
	qNameQryUPTerminalWebhook                   = appdef.NewQName(airPackage, "UPTerminalWebhook")
	qNameCDocUntillPayments                     = appdef.NewQName(airPackage, "UntillPayments")
	qNameCmdInitiateDeactivateWorkspace         = appdef.NewQName(appdef.SysPackage, "InitiateDeactivateWorkspace")
	qNameCmdUpdateUPLocationRates               = appdef.NewQName(airPackage, "UpdateUPLocationRates")
	qNameCmdUpdateUPProfile                     = appdef.NewQName(airPackage, "UpdateUPProfile")
	qNameQryGetAllUPPayouts                     = appdef.NewQName(airPackage, "GetAllUPPayouts")
	qNameQryGetUPPayouts                        = appdef.NewQName(airPackage, "GetUPPayouts")
	qNameQryGetUPInvoiceParties                 = appdef.NewQName(airPackage, "GetUPInvoiceParties")
	qNameQryGetUPFeesOverview                   = appdef.NewQName(airPackage, "GetUPFeesOverview")
	qNameQryGetUPTransactionsOverview           = appdef.NewQName(airPackage, "GetUPTransactionsOverview")
	qNameQryGetUPTransactionReceipts            = appdef.NewQName(airPackage, "GetUPTransactionReceipts")
	qNameQryGetUPTransferInstrument             = appdef.NewQName(airPackage, "GetUPTransferInstrument")
	qNameCmdRetryTransferUPPayout               = appdef.NewQName(airPackage, "RetryTransferUPPayout")
	qNameQryGetUPLocationSubjects               = appdef.NewQName(airPackage, "GetUPLocationSubjects")
	qNameQryIsDirectReseller                    = appdef.NewQName(airPackage, "IsDirectReseller")
	qNameQryGetUPLocationInvoiceParties         = appdef.NewQName(airPackage, "GetUPLocationInvoiceParties")
	qNameQryGetLocationDailyUPReport            = appdef.NewQName(airPackage, "GetLocationDailyUPReport")
	qNameQryGetResellerDailyUPReport            = appdef.NewQName(airPackage, "GetResellerDailyUPReport")
	qNameCDocReseller                           = appdef.NewQName(airPackage, "Reseller")
	qNameQryListIssuedSubscriptionInvoices      = appdef.NewQName(airPackage, "ListIssuedSubscriptionInvoices")
	qNameQryListIssuedSubscriptionResellers     = appdef.NewQName(airPackage, "ListIssuedSubscriptionResellers")
	qNameQryListPaidSubscriptionInvoices        = appdef.NewQName(airPackage, "ListPaidSubscriptionInvoices")
	qNameQryListPaidSubscriptionResellers       = appdef.NewQName(airPackage, "ListPaidSubscriptionResellers")
	qNameQryPaidSubscriptionInvoicesReport      = appdef.NewQName(airPackage, "PaidSubscriptionInvoicesReport")
	qNameCmdVoidUntillPayment                   = appdef.NewQName(airPackage, "VoidUntillPayment")
	qNameQryCreateTap2PaySession                = appdef.NewQName(airPackage, "CreateTap2PaySession")
	qNameQryGetUPLocationRates                  = appdef.NewQName(airPackage, "GetUPLocationRates")
	qNameQryGetDigitalReceipt                   = appdef.NewQName(airPackage, "GetDigitalReceipt")
	qNameQrySendReceiptByEmail                  = appdef.NewQName(airPackage, "SendReceiptByEmail")
	qNameQryUpdateShopperStatement              = appdef.NewQName(airPackage, "UpdateShopperStatement")
	qNameWDocLastNumbers                        = appdef.NewQName(airPackage, "LastNumbers")
	qNameCmdSaveTap2PayPayment                  = appdef.NewQName(airPackage, "SaveTap2PayPayment")
	qNameQryGetResellers                        = appdef.NewQName(airPackage, "GetResellers")
	qNameQryState                               = appdef.NewQName(appdef.SysPackage, "State")
	qNameQryGetAirLocations                     = appdef.NewQName(airPackage, "GetAirLocations")
	qNameQryResellersDashboardSalesMetrics      = appdef.NewQName(airPackage, "ResellersDashboardSalesMetrics")
	qNameQryResellersDashboardBackofficeMetrics = appdef.NewQName(airPackage, "ResellersDashboardBackofficeMetrics")
	qNameCmdProvideCertificatePart              = appdef.NewQName(fiscalcloudPackage, "ProvideCertificatePart")
	qNameCmdProvideCertificate                  = appdef.NewQName(fiscalcloudPackage, "ProvideCertificate")
	qNameQryGetCustomerStatus                   = appdef.NewQName(fiscalcloudPackage, "GetCustomerStatus")
	qNameCmdFiscalizeDocument                   = appdef.NewQName(fiscalcloudPackage, "FiscalizeDocument")
	qNameQryFiscalizationResultStatus           = appdef.NewQName(fiscalcloudPackage, "FiscalizationResultStatus")
	qNameCmdCreateExport                        = appdef.NewQName(fiscalcloudPackage, "CreateExport")
	qNameQryExportStatus                        = appdef.NewQName(fiscalcloudPackage, "ExportStatus")
	qNameQryValidateDocument                    = appdef.NewQName(fiscalcloudPackage, "ValidateDocument")
	qNameCmdAddCustomer                         = appdef.NewQName(fiscalcloudPackage, "AddCustomer")
	qNameCmdUpdateCustomer                      = appdef.NewQName(fiscalcloudPackage, "UpdateCustomer")
	qNameCmdDeactivateCustomer                  = appdef.NewQName(fiscalcloudPackage, "DeactivateCustomer")
	qNameQryGetUPInvoices                       = appdef.NewQName(airPackage, "GetUPInvoices")
	qNameQryGetUPPayoutTransfers                = appdef.NewQName(airPackage, "GetUPPayoutTransfers")
	qNameQryGetAllUPInvoices                    = appdef.NewQName(airPackage, "GetAllUPInvoices")
	qNameQryGetAllUPPayoutTransfers             = appdef.NewQName(airPackage, "GetAllUPPayoutTransfers")
	qNameQryGetDailyUPReports                   = appdef.NewQName(airPackage, "GetDailyUPReports")
	qNameQryGetUPVATTransfers                   = appdef.NewQName(airPackage, "GetUPVATTransfers")
	qNameQryGetUPBeneficiaryVATDebts            = appdef.NewQName(airPackage, "GetUPBeneficiaryVATDebts")
	qNameQryShowBillOnDisplay                   = appdef.NewQName(airPackage, "ShowBillOnDisplay")
	qNameQryShowOrderOnDisplay                  = appdef.NewQName(airPackage, "ShowOrderOnDisplay")
	qNameQryShowStandbyOnDisplay                = appdef.NewQName(airPackage, "ShowStandbyOnDisplay")
	qNameCmdRegisterTempBLOB1d                  = appdef.NewQName(appdef.SysPackage, "RegisterTempBLOB1d")
	qNameQryDownloadBLOBAuthnz                  = appdef.NewQName(appdef.SysPackage, "DownloadBLOBAuthnz")

	// Air roles
	qNameRoleResellersAdmin            = appdef.NewQName(airPackage, "ResellersAdmin")
	qNameRoleUntillPaymentsReseller    = appdef.NewQName(airPackage, "UntillPaymentsReseller")
	qNameRoleUntillPaymentsUser        = appdef.NewQName(airPackage, "UntillPaymentsUser")
	qNameRoleAirReseller               = appdef.NewQName(airPackage, "AirReseller") // Deprecated: use role air.SubscriptionReseller instead
	qNameRoleSubscriptionReseller      = appdef.NewQName(airPackage, "SubscriptionReseller")
	qNameRoleUntillPaymentsTerminal    = appdef.NewQName(airPackage, "UntillPaymentsTerminal")
	qNameRoleUntillPaymentsManager     = appdef.NewQName(airPackage, "UntillPaymentsManager")
	qNameRoleBOReader                  = appdef.NewQName(airPackage, "BOReader")
	qNameResellerPortalDashboardViewer = appdef.NewQName(airPackage, "ResellerPortalDashboardViewer")
	qNameRoleFiscalCloudOnboardSite    = appdef.NewQName(fiscalcloudPackage, "OnboardSite")
)

const (
	field_DeviceProfileWSID     = "DeviceProfileWSID"
	field_ComputersID           = "ComputersID"
	field_RestaurantComputersID = "RestaurantComputersID"
	field_dummy                 = "dummy"
	field_OwnerWSID             = "OwnerWSID"
	airPackage                  = "air"
	untillPackage               = "untill"
	fiscalcloudPackage          = "fiscalcloud"
	untillChargebeeAgentLogin   = "untillchargebeeagent"
	clusterPackage              = "cluster"

	// avoiding import cycle: collection->iauthnzimpl->registry->workspace->collection
	registryPackage = "registry"
)
