-- package consists of schema and resources
-- schema consists of few schema files
SCHEMA Air;

IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS Air;

-- Declare comment to assign it later to definition(s)
COMMENT BackofficeComment "Backoffice Comment";

-- Declare tag to assign it later to definition(s)
TAG BackofficeTag;

-- Declares ROLE
ROLE UntillPaymentsUser;


-- Function which takes sys.TableRow (unnamed param), returns boolean and implemented in WASM module in this package
FUNCTION ValidateRow(TableRow) RETURNS boolean ENGINE WASM;

-- Function which takes named parameter, returns boolean, and implemented in WASM module in this package
FUNCTION ValidateFState(State int) RETURNS boolean ENGINE WASM;


-- TABLE ... OF - declares the inheritance from type or table. PROJECTORS from the base table are not inherted.
TABLE AirTablePlan OF CDOC (
    FState int,
    Name text NOT NULL,
    VerifiableField text NOT NULL VERIFIABLE, -- Verifiable field
    Int1 int DEFAULT 1 CHECK(Int1 >= 1 AND Int2 < 10000),  -- Expressions evaluating to TRUE or UNKNOWN succeed.
    Text1 text DEFAULT "a",
    Int2 int DEFAULT NEXTVAL('sequence'),
    BillID int64 REFERENCES air.bill,
    CheckedField text CHECK "^[0-9]{8}$", -- Field validated by regexp
    CHECK (ValidateRow(this)), -- Unnamed CHECK table constraint. Expressions evaluating to TRUE or UNKNOWN succeed.
    CONSTRAINT StateChecker CHECK (ValidateFState(FState)), -- Named CHECK table constraint
    UNIQUE (FState, Name) -- unnamed UNIQUE table constraint
) WITH Comment=BackofficeComment, Tags=[BackofficeTag]; -- Optional comment and tags


-- Singletones are always CDOC. Error is thrown on attempt to declare it as WDOC or ODOC
TABLE SubscriptionProfile OF SINGLETONE (
    CustomerID text,
    CustomerKind int,
    CompanyName text
);

WORKSPACE MyWorkspace (
    DESCRIPTOR OF NamedType ( -- Workspace descriptor is always SINGLETONE. Error is thrown on attempt to declare it as WDOC or ODOC
        Country text CHECK "^[A-Za-z]{2}$",
        Description text
    );

    -- Declare comments, tags and roles which only available in this workspace
    COMMENT PosComment "Pos Comment";
    TAG PosTag;
    ROLE LocationManager;

    -- Declare rates
    RATE BackofficeFuncRate1 1000 PER HOUR;
    RATE BackofficeFuncRate2 100 PER MINUTE PER IP;

    -- It is only allowed create table if it is defined in this workspace, or added with USE statement
	USE TABLE AirTablePlan;
	USE TABLE SomeSchema.SomeTable;
	USE TABLE Untill.*; 

    TYPE NamedType (
        Name text
    );

    TABLE WsTable OF CDOC, Air.NamedType ( -- Multiple inheritance
        PsName text,
        TABLE Child (
            Number int				
        )
    );	

    -- Functions which are only used by statements within this workspace
    FUNCTION SomeProjectorFunc(Event) RETURNS void ENGINE BUILTIN;
    FUNCTION SomeProjectorFunc2(event sys.Event) RETURNS void ENGINE BUILTIN;
    FUNCTION OrderFunc(Untill.Orders) RETURNS void ENGINE BUILTIN;
    FUNCTION Order2Func(Untill.Orders, Untill.PBill) RETURNS void ENGINE BUILTIN;
    FUNCTION QueryFunc() RETURNS text ENGINE BUILTIN;
    FUNCTION Qiery2Func(Untill.Orders, Untill.PBill) RETURNS text ENGINE BUILTIN;

    -- Projectors can only be declared in workspaces. Function can only take sys.Event as argument and return void.
    PROJECTOR ON COMMAND Air.Orders2 AS SomeProjectorFunc;
    PROJECTOR ON COMMAND ARGUMENT NamedType AS Air.SomeProjectorFunc2;
    PROJECTOR ON INSERT Air.AirTablePlan AS SomeProjectorFunc;
    PROJECTOR ON INSERT OR UPDATE IN (Air.AirTablePlan, WsTable) AS SomeProjectorFunc;
    PROJECTOR ON UPDATE Air.AirTablePlan AS SomeProjectorFunc;
    PROJECTOR ON UPDATE OR INSERT Air.AirTablePlan AS SomeProjectorFunc;
    PROJECTOR ON ACTIVATE Air.AirTablePlan AS SomeProjectorFunc; -- Triggered when Article is activated
    PROJECTOR ON ACTIVATE OR DEACTIVATE Air.AirTablePlan AS SomeProjectorFunc; -- Triggered when Article is activated or deactivated
    PROJECTOR ApplyUPProfile ON COMMAND IN (Air.Orders2, Air.Orders3) AS Air.SomeProjectorFunc;

    -- Commands can only be declared in workspaces
    COMMAND Orders2(Untill.Orders) AS OrderFunc;
    
    -- Command with declared Comment, Tags and Rate
    COMMAND Orders3(Order Untill.Orders, Untill.PBill) AS Order2Func WITH 
        Comment=Air.PosComment, 
        Tags=[BackofficeTag, Air.PosTag],
        Rate=BackofficeFuncRate1; 

    -- Qieries can only be declared in workspaces
    QUERY Query1 RETURNS text AS QueryFunc;
    QUERY _Query1() RETURNS text AS QueryFunc WITH Comment=Air.PosComment, Tags=[BackofficeTag, Air.PosTag];
    QUERY Query2(Order Untill.Orders, Untill.PBill) RETURNS text AS Qiery2Func;


    -- ACLs
    GRANT ALL ON ALL TABLES WITH TAG untill.Backoffice TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE Untill.Orders TO LocationUser;
    GRANT EXECUTE ON COMMAND Orders TO LocationUser;
    GRANT EXECUTE ON QUERY TransactionHistory TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;


    -- VIEW generated by PROJECTOR. 
    -- Primary Key must be declared in View.
    VIEW XZReports(
        Year int32,
        Month int32, 
        Day int32, 
        Kind int32, 
        Number int32, 
        XZReportWDocID id,
        PRIMARY KEY ((Year), Month, Day, Kind, Number)
    ) AS RESULT OF Air.UpdateXZReportsView;


);

ABSTRACT WORKSPACE AWorkspace (
    -- Abstract workspaces cannot be created
);

WORKSPACE MyWorkspace1 OF AWorkspace (
    -- Inherits everything declared in AWorkspace
);
