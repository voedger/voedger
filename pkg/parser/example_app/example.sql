-- package consists of schema and resources
-- schema consists of few schema files
SCHEMA air;

IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS air;

-- Declare comment to assign it later to definition(s)
COMMENT BackofficeComment "Backoffice Comment";

-- Declare tag to assign it later to definition(s)
TAG BackofficeTag;

-- Declares ROLE
ROLE UntillPaymentsUser;

-- TABLE ... OF - declares the inheritance from type or table. PROJECTORS from the base table are not inherted.
TABLE AirTablePlan INHERITS CDoc (
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
    UNIQUE (FState, Name), -- unnamed UNIQUE table constraint
    TABLE AirTablePlanItem (
        TableNo int,
        Chairs int
    )
) WITH Comment=BackofficeComment, Tags=[BackofficeTag]; -- Optional comment and tags


-- Singletones are always CDOC. Error is thrown on attempt to declare it as WDOC or ODOC
TABLE SubscriptionProfile INHERITS Singleton (
    CustomerID text,
    CustomerKind int,
    CompanyName text
);

-- Package-level extensions
EXTENSION ENGINE WASM (

    -- Function which takes sys.TableRow (unnamed param), returns boolean and implemented in WASM module in this package
    FUNCTION ValidateRow(TableRow) RETURNS boolean;

    -- Function which takes named parameter, returns boolean, and implemented in WASM module in this package
    FUNCTION ValidateFState(State int) RETURNS boolean;

);

WORKSPACE MyWorkspace (
    DESCRIPTOR OF TypeWithName ( -- Workspace descriptor is always SINGLETONE. Error is thrown on attempt to declare it as WDOC or ODOC
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

    TYPE TypeWithName (
        Name text
    );
    TYPE TypeWithKind (
        Kind int
    );
    TYPE SubscriptionEvent (
        Origin text,
        Data text
    );


    TABLE WsTable INHERITS CDoc OF air.TypeWithName, TypeWithKind ( -- Multiple types
        PsName text,
        TABLE Child (
            Number int				
        )
    );	

    -- Workspace-level extensions 
    EXTENSION ENGINE BUILTIN (

        -- Projector can only be declared in workspace.
        -- A builtin function OrdersCountProjector must exist in package resources.
        -- TARGET - lists all QNames for which Intets are generated (QName of Entity or Storage)
        -- USE - lists all QNames for which Get/Read operations are done (QName of Entity or Storage). 
        --      (no need to specify in USES when already listed in TARGET)
        PROJECTOR CountOrders ON COMMAND air.Orders MAKES air.OrdersCountView;
        
        -- Projector triggered by command argument SubscriptionProfile which is a Storage
        -- Projector uses sys.HTTPStorage
        PROJECTOR UpdateSubscriptionProfile ON COMMAND ARGUMENT SubscriptionEvent USES sys.HTTPStorage;

        -- Projectors triggered by CUD operations
        PROJECTOR AirPlanThumbnailGen ON INSERT air.AirTablePlan MAKES AirPlanThumbnails;
        PROJECTOR UpdateDashboard ON COMMAND IN (air.Orders, air.Orders2) MAKES DashboardView;
        PROJECTOR UpdateActivePlans ON ACTIVATE OR DEACTIVATE air.AirTablePlan MAKES ActiveTablePlansView;
        
        -- Some projector which sends E-mails and performs HTTP queries
        PROJECTOR NotifyOnChanges ON INSERT OR UPDATE IN (air.AirTablePlan, WsTable) USES sys.HTTPStorage MAKES sys.SendMailStorage;

        -- Commands can only be declared in workspaces
        COMMAND Orders(Untill.Orders);
        
        -- Command with declared Comment, Tags and Rate
        COMMAND Orders2(Order Untill.Orders, Untill.PBill) WITH 
            Comment=air.PosComment, 
            Tags=[BackofficeTag, air.PosTag],
            Rate=BackofficeFuncRate1; 

        -- Qieries can only be declared in workspaces
        QUERY Query1 RETURNS text;
        QUERY _Query1() RETURNS text WITH Comment=air.PosComment, Tags=[BackofficeTag, air.PosTag];
        QUERY Query2(Order Untill.Orders, Untill.PBill) RETURNS text;
    );

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
        XZReportWDocID id NOT NULL,
        PRIMARY KEY ((Year), Month, Day, Kind, Number)
    ) AS RESULT OF air.UpdateDashboard;

    VIEW OrdersCountView(
        Year int, -- same as int32
        Month int32, 
        Day sys.int32, -- same as int32
        Qnantity int32,
        SomeField int32,
        PRIMARY KEY ((Year), Month, Day)
    ) AS RESULT OF CountOrders;

);

ABSTRACT WORKSPACE AWorkspace (
    -- Abstract workspaces cannot be created
);

WORKSPACE MyWorkspace1 OF AWorkspace (
    -- Inherits everything declared in AWorkspace
    POOL OF WORKSPACE MyPool ()
);
