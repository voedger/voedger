-- package consists of schema and resources
-- schema consists of few schema files
SCHEMA Air;


// TODO: How we handle this?
IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS Air;

COMMENT BackofficeComment "Backoffice Comment";
TAG BackofficeTag;

-- Void function, no parameters
FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;

-- Function with string as a return type
FUNCTION MyTableValidator2(sys.TableRow) RETURNS string ENGINE WASM;

// TODO: better comments


FUNCTION MyTableValidator3(param1 sys.TableRow, bbb.string) RETURNS ccc.TableRow ENGINE WASM;

FUNCTION SomeProjectorFunc() RETURNS void ENGINE BUILTIN;
FUNCTION FillUPProfile(sys.Event) RETURNS void ENGINE BUILTIN;
FUNCTION SomeCmdFunc() RETURNS void ENGINE BUILTIN;

ROLE UntillPaymentsUser;

-- Comment for table
TABLE AirTablePlan OF CDOC (
    FState int,
    Name text NOT NULL,
    VerifiableField text NOT NULL VERIFIABLE, -- Comment
    Int1 int DEFAULT 1,
    Text1 text DEFAULT "a",
    Int2 int DEFAULT NEXTVAL('sequence'),
    BillID int64 REFERENCES air.bill,
    CheckedField text CHECK "^[0-9]{8}$",
    UNIQUE fstate, name
) WITH Comment=BackofficeComment, Tags=[BackofficeTag];

WORKSPACE MyWorkspace (

    COMMENT PosComment "Pos Comment";
    TAG PosTag;

	USE TABLE SomeSchema.SomeTable;
	USE TABLE AirTablePlan;
	USE TABLE Untill.*; 

    ROLE LocationManager;

    FUNCTION OrderFunc(Untill.Orders) RETURNS void ENGINE BUILTIN;
    FUNCTION Order2Func(Untill.Orders, Untill.PBill) RETURNS void ENGINE BUILTIN;
    FUNCTION QueryFunc() RETURNS text ENGINE BUILTIN;
    FUNCTION Qiery2Func(Untill.Orders, Untill.PBill) RETURNS text ENGINE BUILTIN;

    PROJECTOR ON COMMAND Air.CreateUPProfile AS SomeProjectorFunc;
    PROJECTOR ON COMMAND ARGUMENT Untill.QNameOrders AS Air.SomeProjectorFunc;
    PROJECTOR ON INSERT Untill.Bill AS SomeProjectorFunc;
    PROJECTOR ON INSERT OR UPDATE Untill.Bill AS SomeProjectorFunc;
    PROJECTOR ON UPDATE Untill.Bill AS SomeProjectorFunc;
    PROJECTOR ON UPDATE OR INSERT Untill.Bill AS SomeProjectorFunc;
    PROJECTOR ApplyUPProfile ON COMMAND IN (Air.CreateUPProfile, Air.UpdateUPProfile) AS Air.FillUPProfile;

    COMMAND Orders AS SomeCmdFunc;
    COMMAND _Orders() AS SomeCmdFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
    COMMAND Orders2(Untill.Orders) AS OrderFunc;
    COMMAND Orders3(Order Untill.Orders, Untill.PBill) AS Order2Func;

    QUERY Query1 RETURNS text AS QueryFunc;
    QUERY _Query1() RETURNS text AS QueryFunc WITH Comment=Air.PosComment, Tags=[Tag1, Air.Tag2];
    QUERY Query2(Order Untill.Orders, Untill.PBill) RETURNS text AS Qiery2Func;


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
        XZReportWDocID id
    ) AS RESULT OF Air.UpdateXZReportsView
    WITH 
        PrimaryKey="(Year), Month, Day, Kind, Number",
        Comment=PosComment;


    RATE BackofficeFuncRate1 1000 PER HOUR;
    RATE BackofficeFuncRate2 100 PER MINUTE PER IP;

);
