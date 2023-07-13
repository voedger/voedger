-- package consists of schema and resources
-- schema consists of few schema files
SCHEMA main;

IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS air;

-- Declare comment to assign it later to definition(s)
COMMENT BackofficeComment "Backoffice Comment";

-- Declare tag to assign it later to definition(s)
TAG BackofficeTag;

-- Declares ROLE
ROLE UntillPaymentsUser;

TABLE NestedTable INHERITS CRecord (
    ItemName text
);

TABLE ScreenGroup INHERITS CDoc();

-- TABLE ... OF - declares the inheritance from type or table. PROJECTORS from the base table are not inherted.
TABLE TablePlan INHERITS CDoc (
    FState int,
    Name text NOT NULL,
    VerifiableField text NOT NULL VERIFIABLE, -- Verifiable field
    Int1 int DEFAULT 1 CHECK(Int1 >= 1 AND Int2 < 10000),  -- Expressions evaluating to TRUE or UNKNOWN succeed.
    Text1 text DEFAULT "a",
    Int2 int DEFAULT NEXTVAL('sequence'),
    ScreenGroupRef ref(ScreenGroup), 
    AnyTableRef ref,
    FewTablesRef ref(ScreenGroup, TablePlan) NOT NULL,
    CheckedField text CHECK "^[0-9]{8}$", -- Field validated by regexp
    CHECK (ValidateRow(this)), -- Unnamed CHECK table constraint. Expressions evaluating to TRUE or UNKNOWN succeed.
    CONSTRAINT StateChecker CHECK (ValidateFState(FState)), -- Named CHECK table constraint
    -- UNIQUE (FState, Name), -- unnamed UNIQUE table constraint
    UNIQUEFIELD Name, -- deprecated. For Air backward compatibility only
    TableItems TABLE TablePlanItem (
        TableNo int,
        Chairs int
    ),
    items NestedTable,
    ExcludedTableItems TablePlanItem
) WITH Comment=BackofficeComment, Tags=(BackofficeTag); -- Optional comment and tags


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
    DESCRIPTOR OF air.TypeWithName ( -- Workspace descriptor is always SINGLETONE. Error is thrown on attempt to declare it as WDOC or ODOC
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
	USE TABLE SomeSchema.SomeTable;
	USE TABLE untill.*; 

    TYPE TypeWithKind (
        Kind int
    );
    TYPE SubscriptionEvent (
        Origin text,
        Data text
    );


    TABLE WsTable INHERITS CDoc OF air.TypeWithName, TypeWithKind ( -- Multiple types
        PsName text,
        items TABLE Child (
            Number int				
        )
    );	

    -- Workspace-level extensions 
    EXTENSION ENGINE BUILTIN (

        -- Projector can only be declared in workspace.
        -- A builtin function OrdersCountProjector must exist in package resources.
        -- INTENTS - lists all storage keys, projector generates intents for
        -- STATE - lists all storage keys, projector reads state from
        --      (key consist of Storage Qname, and Entity name, when required by storage)
        --      (no need to specify in STATE when already listed in INTENTS)
        PROJECTOR CountOrders 
            ON COMMAND Orders 
            INTENTS(View air.OrdersCountView);
        
        -- Projector triggered by command argument SubscriptionProfile which is a Storage
        -- Projector uses sys.HTTPStorage
        PROJECTOR UpdateSubscriptionProfile 
            ON COMMAND ARGUMENT SubscriptionEvent 
            STATE(sys.Http, AppSecrets);

        -- Projectors triggered by CUD operations
        -- SYNC means that projector is synchronous 
        SYNC PROJECTOR TablePlanThumbnailGen 
            ON INSERT TablePlan 
            INTENTS(View TablePlanThumbnails);

        PROJECTOR UpdateDashboard 
            ON COMMAND IN (Orders, Orders2) 
            INTENTS(View DashboardView);

        PROJECTOR UpdateActivePlans 
            ON ACTIVATE OR DEACTIVATE TablePlan 
            INTENTS(View ActiveTablePlansView);
        
        -- Some projector which sends E-mails and performs HTTP queries
        PROJECTOR NotifyOnChanges 
            ON INSERT OR UPDATE IN (TablePlan, WsTable) 
            STATE(Http, AppSecrets)
            INTENTS(SendMail, View air.NotificationsHistory);

        -- Commands can only be declared in workspaces
        -- Command can have optional argument and/or unlogged argument
        -- Command can return TYPE
        COMMAND Orders(air.Order, UNLOGGED air.TypeWithName) RETURNS air.Order;
        
        -- Command can return void (in this case `RETURNS void` may be omitted)
        COMMAND Orders2(air.Order) RETURNS void;

        -- Command with declared Comment, Tags and Rate
        COMMAND Orders4(UNLOGGED air.Order) WITH 
            Comment=PosComment, 
            Tags=(BackofficeTag, PosTag),
            Rate=BackofficeFuncRate1; 

        -- Qieries can only be declared in workspaces
        QUERY Query1 RETURNS void;
        QUERY _Query1() RETURNS air.Order WITH Comment=PosComment, Tags=(BackofficeTag, PosTag);
        QUERY Query2(air.Order) RETURNS air.Order;
    );

    -- ACLs
    GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE Orders TO LocationUser;
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
