-- Copyright (c) 2020-present unTill Pro, Ltd.

SCHEMA air;

IMPORT SCHEMA github.com/untillpro/airs-bp3/packages/untill

WORKSPACE Restaurant (

    -- Roles
    ROLE UntillPaymentsUser;
    ROLE LocationManager;
    ROLE LocationUser;


    -- Tables
    USE TABLE untill.*; --Every workspace Restaurant has all tables from schema `untill`

    -- Collection is applied to all tables with tag "sys.Collection"
    TAG ON TAG "Backoffice" IS "sys.Collection"

    --SYNONIM uarticles FOR untill.articles  --later


    ---- TO be added later:
    ---- Misc Functions: e.g. to use in the inline CHECKS
    ---- Arguments and return value used to work with State and Intents when calling this extension
    FUNCTION ApproxEqual(param1 float, param2 float) RETURNS boolean ENGINE BUILTIN;


    --- Remove procedure, declare arguments and results in CHECK, COMMAND and QUERY
    -- CHECKS
    VALIDATOR MyBillValidator AS ENGINE BUILTIN; -- same as MyBillValidator(sys.TableRow)
    VALIDATOR MyFieldsValidator(fieldA text, fieldB text) AS WasmFuncName ENGINE BUILTIN; --

    CHECK ON TABLE untill.bill IS MyBillValidator;
    CHECK ON TABLE untill.bill(name, pcname) IS MyFieldsValidator;

    -- PROJECTORS

    PROJECTOR FillUPProfile() AS ENGINE WASM; -- Same as FillUPProfile(sys.Event)
    PROJECTOR FillUPProfile(sys.Event) AS WasmFuncName ENGINE WASM;
    PROJECTOR ON EVENT WITH TAG Backoffice IS FillUPProfile;
    PROJECTOR ON EVENT air.CreateUPProfile AS WasmFuncName ENGINE WASM;
    PROJECTOR ON EVENT IN (air.CreateUPProfile, air.UpdateUPProfile) IS FillUPProfile;

    -- COMMANDS
    COMMAND Orders(untill.orders) AS ENGINE BUILTIN; -- Return is optional = same as RETURNS void;
    COMMAND Pbill(untill.pbill) RETURNS PbillResult AS PbillImpl ENGINE BUILTIN;
    COMMAND LinkDeviceToRestaurant(LinkDeviceToRestaurantParams) RETURNS void IS somepackage.MiscFunc;

    -- DECLARE RATE BackofficeFuncRate AS 100 PER MINUTE PER IP;    <- rejected by NNV :)
    RATE BackofficeFuncRate AS 100 PER MINUTE PER IP;
    Comment BackofficeDescription AS "This is a backoffice table";

    -- QUERIES
    QUERY TransactionHistory(TransactionHistoryParams) RETURNS TransactionHistoryResult[] ENGINE WASM
        WITH Rate=BackofficeFuncRate, Comment='Transaction History'

    COMMENT ON QUERY TransactionHistory IS 'Transaction History';
    COMMENT ON QUERY WITH TAG Backoffice IS 'Transaction History';
    COMMENT ON QUERY IN (TransactionHistory, ...) IS 'Transaction History';

    RATE ON QUERY TransactionHistory IS BackofficeFuncRate;
    RATE ON QUERY TransactionHistory AS 101 PER MINUTE PER IP;


    QUERY QueryResellerInfo(reseller_id text) RETURNS QueryResellerInfoResult ENGINE WASM;


    -- ACL
    GRANT ALL ON TABLE WITH TAG untill.Backoffice TO LocationManager
    GRANT INSERT,UPDATE ON TABLE WITH TAG sys.ODoc TO LocationUser
    GRANT SELECT ON TABLE untill.orders TO LocationUser
    GRANT EXECUTE ON COMMAND PBill TO LocationUser
    GRANT EXECUTE ON COMMAND Orders TO LocationUser
    GRANT EXECUTE ON QUERY TransactionHistory TO LocationUser


    TYPE TransactionHistoryParams AS (
        BillIDs text NOT NULL,
        EventTypes text NOT NULL,
    )

    TYPE TransactionHistoryResult AS (
        Offset offset NOT NULL,
        EventType int64 NOT NULL,
        Event text NOT NULL,
    )


    -- dashboard: hourly sales
    VIEW HourlySalesView(yyyymmdd, hour, total, count) AS
    SELECT
        working_day as yyyymmdd,
        EXTRACT(hour from ord_datetime) as hour,
        SUM(price * quantity) as total,
        SUM(quantity) as count
        from untill.orders
            join order_item on order_item.id_orders=orders.id
        group by working_day, hour
    WITH Key='(yyyymmdd), hour)';

    -- dashboard: daily categories
    VIEW DailyCategoriesView(yyyymmdd PK, id_category, total) A
    SELECT
        working_day as yyyymmdd,
        id_category,
        SUM(price * quantity) as total,
        from untill.orders
            join order_item on order_item.id_orders = orders.id
            join articles on id_articles = articles.id
            join department on id_departments = articles.id_department
            join food_group on id_food_group = department.id_food_group
        group by working_day, id_category

    TYPE LinkDeviceToRestaurantParams AS (
        deviceToken text not null,
        deviceName text not null,
        deviceProfileWSID text not null,
    )


    TABLE Restaurant OF SINGLETONE (
        WorkStartTime text,
        DefaultCurrency int64,
        NextCourseTicketLayout int64,
        TransferTicketLayout int64,
        DisplayName text,
        Country text,
        City text,
        ZipCode text,
        Address text,
        PhoneNumber text,
        VATNumber text,
        ChamberOfCommerce text,
    )

    TYPE WriteResellerInfoParams AS (
        reseller_id text,
        reseller_phone text,
        reseller_company text,
        reseller_email text,
        reseller_website text
    )

    TYPE QueryResellerInfoResult AS (
        reseller_phone text,
        reseller_company text,
        reseller_email text,
        reseller_website text
    );


    VIEW TablesOverview(
        partitionKey int32, tableNumber int32, tablePart text, wDocID id,
        PRIMARY KEY((partitionKey), tableno, table_part)
    ) as select
        2 as partitionKey,
        tableno as tableNumber,
        table_part as tablePart,
        sys.ID as id
    from untill.bill

    VIEW TransactionHistory(wDocID id, offs offset, PRIMARY KEY((id), offs)) AS
    select id, sys.Offset from untill.bill
    union all select id_bill, sys.Offset from orders
    union all select id_bill, sys.Offset from pbill ;




    -- XZ Reports
    TYPE CreateXZReportParams AS(
        Kind int32,
        Number int32,
        WaiterID id,
        from int64,
        till int64
    )

    VIEW XZReports(
        Year int32,
        Month int32,
        Day int32,
        Kind int32,
        Number int32,
        XZReportWDocID id,
        PRIMARY KEY((Year), Month, Day, Kind, Number)
    ) AS RESULT OF UpdateXZReportsView



) -- WORKSPACE Restaurant

WORKSPACE Resellers {

    ROLE ResellersAdmin;

    WORKSPACE Reseller {
        ROLE UntillPaymentsReseller;
        ROLE AirReseller;
        USE Table PaymentsProfile
    }
}

TEMPLATE demo OF WORKSPACE air.Restaurant WITH SOURCE wsTemplate_demo;
TEMPLATE resdemo OF WORKSPACE untill.Resellers WITH SOURCE wsTemplate_demo_resellers;


-- ??? indexes: BillDates, OrderDates
-- provideQryIssueLinkDeviceToken


-- Subscription Query functions:
-- - QueryResellerInfo
-- - FindRestaurantSubscription
-- - EstimatePlan
-- - GetHostedPage
-- - UpdateSubscriptionDetails
-- - UpdatePaymentMethodHostedPage
-- - CancelSubscription
-- - VaidateVAT
-- - EstimateUpgradePlan
-- - QryCompleteTrialPeriod

