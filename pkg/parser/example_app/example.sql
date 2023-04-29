SCHEMA air;

IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS Air;

-- Declare comment to assign it later to definition(s)
COMMENT BackofficeComment "Backoffice Comment";

-- Declare tag to assign it later to definition(s)
TAG BackofficeTag;

-- Function which has no arguments, returns nothing and implemented in core
FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;

-- Function which takes sys.TableRow (unnamed param), returns text and implemented in WASM module in this package
FUNCTION MyTableValidator2(TableRow) RETURNS text ENGINE WASM;

-- Function which takes two named parameters returns sys.TableRow, and implemented in WASM module in this package
FUNCTION MyTableValidator3(param1 sys.TableRow, param2 string) RETURNS sys.TableRow ENGINE WASM;

-- Declares ROLE
ROLE UntillPaymentsUser;

-- TABLE ... OF - declares the inheritance from type or table. PROJECTORS from the base table are not inherted.
TABLE AirTablePlan OF CDOC (
    FState int,
    Name text NOT NULL,
    VerifiableField text NOT NULL VERIFIABLE, -- Verifiable field
    Int1 int DEFAULT 1, 
    Text1 text DEFAULT "a",
    Int2 int DEFAULT NEXTVAL('sequence'),
    BillID int64 REFERENCES air.bill,
    CheckedField text CHECK "^[0-9]{8}$", -- Field validated by regexp
    UNIQUE (fstate, name)
) WITH Comment=BackofficeComment, Tags=[BackofficeTag]; -- Optional comment and tags


-- Singletones are always CDOC. Error is thrown on attempt to declare it as WDOC or ODOC
TABLE SubscriptionProfile OF SINGLETONE (
    CustomerID text,
    CustomerKind int,
    CompanyName text
);

WORKSPACE MyWorkspace (
    DESCRIPTOR OF NamedType ( -- Workspace descriptor is always SINGLETONE. Error is thrown on attempt to declare it as WDOC or ODOC
        Description text
    );

    -- Declare comments, tags and roles which only available in this workspace
    COMMENT PosComment "Pos Comment";
    TAG PosTag;
    ROLE LocationManager;

    -- Tables which can be created in this workspace
	USE TABLE AirTablePlan;
	USE TABLE SomeSchema.SomeTable;
	USE TABLE Untill.*; 


    -- Projectors can only be declared in workspaces
    PROJECTOR ON COMMAND Air.CreateUPProfile AS SomeFunc;
    PROJECTOR ON COMMAND ARGUMENT Untill.QNameOrders AS SomeSchema.SomeFunc2;
    PROJECTOR ON INSERT Untill.Bill AS SomeFunc;
    PROJECTOR ON INSERT OR UPDATE Untill.Bill AS SomeFunc;
    PROJECTOR ON UPDATE Untill.Bill AS SomeFunc;
    PROJECTOR ON UPDATE OR INSERT Untill.Bill AS SomeFunc;
    PROJECTOR ApplyUPProfile ON COMMAND IN (Air.CreateUPProfile, Air.UpdateUPProfile) AS Air.FillUPProfile;

    -- Commands can only be declared in workspaces
    COMMAND Orders AS PbillFunc;
    COMMAND _Orders() AS PbillFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
    COMMAND Orders2(Untill.Orders) AS PbillFunc;
    COMMAND Orders3(Order Untill.Orders, Untill.PBill) AS PbillFunc;

    -- Qieries can only be declared in workspaces
    QUERY Query1 RETURNS QueryResellerInfoResult AS PbillFunc;
    QUERY _Query1() RETURNS Air.QueryResellerInfoResult AS PbillFunc WITH Comment=Air.PosComment, Tags=[Tag1, Air.Tag2];
    QUERY Query2(Untill.Orders) RETURNS QueryResellerInfoResult AS PbillFunc;
    QUERY Query3(Order Untill.Orders, Untill.PBill) RETURNS QueryResellerInfoResult AS PbillFunc;


    -- ACLs
    GRANT ALL ON ALL TABLES WITH TAG untill.Backoffice TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE Untill.Orders TO LocationUser;
    GRANT EXECUTE ON COMMAND Orders TO LocationUser;
    GRANT EXECUTE ON QUERY TransactionHistory TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    TABLE WsTable OF CDOC, Air.SomeType (
        PsName text,
        TABLE Child (
            Number int				
        )
    );	

    VIEW XZReports(
        Year int32,
        Month int32, 
        Day int32, 
        Kind int32, 
        Number int32, 
        XZReportWDocID id,
        PRIMARY KEY ((Year), Month, Day, Kind, Number)
    ) AS RESULT OF Air.UpdateXZReportsView
    WITH Comment=PosComment;


    RATE BackofficeFuncRate1 1000 PER HOUR;
    RATE BackofficeFuncRate2 100 PER MINUTE PER IP;

);