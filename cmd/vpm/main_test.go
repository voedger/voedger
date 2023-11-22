package main

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	tempDir := t.TempDir()
	appContent := `
IMPORT SCHEMA 'github.com/untillpro/airs-scheme/bp3' AS untill;

APPLICATION test (
	USE untill;
);
`

	sqlContent := `
	--IMPORT SCHEMA 'github.com/untillpro/airs-scheme/bp3' AS untill;
	
	TABLE ProformaPrinted INHERITS ODoc (
		Number int32 NOT NULL,
		UserID ref(untill.untill_users) NOT NULL,
		Timestamp int64 NOT NULL,
		BillID ref(untill.bill) NOT NULL
	);
	
	TABLE NextNumbers INHERITS Singleton (
		NextPBillNumber int32
	);
	
	-- TODO: mandatory fields??? email aor phone?
	TABLE ResellerSubscriptionsProfile INHERITS Singleton (
		Company text,
		ContactEmail text,
		ContactPhone text,
		ContactWebsite text,
		ContactAddress text(1024),
		CustomerID text,
		ContactWhatsappNumber text
	);
	
	TABLE SubscriptionProfile INHERITS Singleton (
		SubscriptionResourceVersion int64,
		SubscriptionID text NOT NULL,
		Status text(30),
		NumberOfDevices int32,
		NextBillingAt int64,
		Currency text(3),
		Price int64,
		CardResourceVersion int64,
		PaymentMethod text,
		LastFour text,
		CancelledAt int64,
		CancelScheduleCreated bool,
		CancelReason text,
		BillingPeriod text,
		AirCluster text,
	   TrialEnd int64
	);
	
	TABLE UPProfile INHERITS Singleton (
		LegalName text NOT NULL,
		ProfileKind int32 NOT NULL,
		LegalEntityID text,
		TransferInstrumentID text,
		AccountHolderID text,
		BalanceAccountID text,
		BusinessLineID text,
		StoreID text,
		Error text(1024),
		ApiToken text(1024),
		Status int32,
		TerminalWebhookToken text(1024),
		ScheduledPayoutSweepID text,
		ScheduledPayoutStatus int32,
		ScheduledPayoutError text(1024),
		SplitConfigurationID text
	);
	
	TABLE Reseller INHERITS Singleton (
		Name text NOT NULL,
		Country text NOT NULL,
		Timezone text(50)
	);
	
	TABLE Resellers INHERITS Singleton ();
	
	TABLE UntillPayments INHERITS Singleton (
		Name text NOT NULL,
		ResellerBalanceAccount text NOT NULL,
		Rate int64 NOT NULL,
		RateCC float32,
		Timezone text(50)
	);
	
	TABLE UPTransfer INHERITS WDoc (
		Type int32 NOT NULL,
		Currency text(3),
		Amount int64 NOT NULL,
		Description text(1024),
		TransferID text,
		Error text(1024)
	);
	
	TABLE UntillPayment INHERITS WDoc (
		ReqOffset int64,
		ResOffset int64,
		Result int32,
		Error text(1024)
	);
	
	TABLE DailyUPReport INHERITS WDoc (
	   ReportURL text,
	   FileName text,
	   Parsed bool
	);
	
	TABLE WorkspaceDailyUPReport INHERITS ODoc (
	   Year int32,
	   Month int32,
	   Day int32,
	   BlobID int64,
	   ValueDates TABLE WorkspaceDailyReportValueDate (
	       Year int32,
	       Month int32,
	       Day int32
	   )
	);
	
	TABLE UPPayout INHERITS WDoc (
	   Created int64,
	   From text,
	   Till text,
	   Number int64,
	   Currency text,
	   Type int32,
	   PaymentsAmount int64,
	   ProcessingUFees int64,
	   ProcessingRFees int64,
	   AcquirerUFees int64,
	   AcquirerRFees int64,
	   UFeesVAT int64,
	   RFeesVAT int64,
	   PaymentFeesTotal int64,
	   PaymentFeesInterchange int64,
	   PaymentFeesSchemeFee int64,
	   PaymentFeesAcquirerMarkup int64,
	   PaymentFeesBlendCommission int64,
	   PayoutAmount int64
	);
	
	TABLE UserProfile INHERITS Singleton (
		ResellerID text,      -- deprecated
		ResellerPhone text,   -- deprecated
		ResellerCompany text, -- deprecated
		ResellerEmail text,   -- deprecated
		ResellerWebsite text, -- deprecated
		Email text VERIFIABLE,
		Name text,
		Country text
	);
	
	TABLE Restaurant INHERITS Singleton (
		WorkStartTime text,
		DefaultCurrency int64,
		NextCourseTicketLayout int64,
		TransferTicketLayout int64,
		DisplayName text,
	
		-- https://dev.untill.com/projects/#!613072
		Country text,
		City text,
		ZipCode text,
		Address text,
		PhoneNumber text,
		VATNumber text,
		ChamberOfCommerce text,
	
		-- https://dev.untill.com/projects/#!617416
		-- 0 - not specified, 1 - false, 2 - true
		UseSalesAreas int32,
		UsePreparationAreas int32,
		UseCourses int32,
		UseTablePlans int32,
	
		-- https://dev.untill.com/projects/#!628926
		StartGuideState int32,
	
		-- https://dev.untill.com/projects/#!629366
		HappyHoursPeriod int64,
	
		-- https://dev.untill.com/projects/#!630536
		DefaultPriceID int64,
		DefaultSpaceID int64,
	
		-- https://dev.untill.com/projects/#!632895
		XZReportTicketLayout ref,
	
		-- https://dev.untill.com/projects/#!632897
		LangOfService text,
	
		-- https://dev.untill.com/projects/#!635708
		BillTicketLayout ref,
		OrderTicketLayout ref,
		UntillPaymentsLocationWSID int64,
		UntillPaymentsToken text(1024),
	
		-- https://dev.untill.com/projects/#!640327
		EnableUntillPayments bool,
	
		-- https://dev.untill.com/projects/#!642996
		AutoLogoff int64,
	
		-- https://dev.untill.com/projects/#!649518
		LocationType int32
	);
	
	TABLE XZReport INHERITS WDoc (
		Kind int32 NOT NULL,
		Time int64 NOT NULL,
		Number int32 NOT NULL,
		WaiterID ref NOT NULL,
		From int64,
		Till int64,
		StartOffset int64,
		EndOffset int64,
		ReportDataID ref,
		TicketDataID ref,
		Status int32 NOT NULL
	);
	`
	goModContent := `
module github.com/untillpro/airs-bp3

go 1.21

require (
	github.com/untillpro/airs-scheme v0.0.0-20231030150820-b4eed4c88d05
)
`
	goModPath := path.Join(tempDir, "go.mod")
	sqlPath := path.Join(tempDir, "schema.sql")
	appPath := path.Join(tempDir, "app.sql")

	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	require.NoError(err)

	err = os.WriteFile(sqlPath, []byte(sqlContent), 0644)
	require.NoError(err)

	err = os.WriteFile(appPath, []byte(appContent), 0644)
	require.NoError(err)

	err = execRootCmd([]string{"vpm", "compile", fmt.Sprintf(" --C=%s", tempDir)}, "1.0.0")
	require.NoError(err)
}
