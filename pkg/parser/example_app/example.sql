SCHEMA air;

IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS Air;

COMMENT BackofficeComment "Backoffice Comment";
TAG BackofficeTag;

-- Comment for function
FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
FUNCTION MyTableValidator2(TableRow) RETURNS string ENGINE WASM;
FUNCTION MyTableValidator3(param1 aaa.TableRow, bbb.string) RETURNS ccc.TableRow ENGINE WASM;

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
    UNIQUE (fstate, name)
) WITH Comment=BackofficeComment, Tags=[BackofficeTag];

WORKSPACE MyWorkspace (
    DESCRIPTOR OF NamedType ( -- Workspace descriptor is always CDOC and SINGLETONE. Error is thrown on attempt to declare it as WDOC or ODOC
        Description text
    );

    COMMENT PosComment "Pos Comment";
    TAG PosTag;

	USE TABLE SomeSchema.SomeTable;
	USE TABLE AirTablePlan;
	USE TABLE Untill.*; 

    ROLE LocationManager;

    PROJECTOR ON COMMAND Air.CreateUPProfile AS SomeFunc;
    PROJECTOR ON COMMAND ARGUMENT Untill.QNameOrders AS SomeSchema.SomeFunc2;
    PROJECTOR ON INSERT Untill.Bill AS SomeFunc;
    PROJECTOR ON INSERT OR UPDATE Untill.Bill AS SomeFunc;
    PROJECTOR ON UPDATE Untill.Bill AS SomeFunc;
    PROJECTOR ON UPDATE OR INSERT Untill.Bill AS SomeFunc;
    PROJECTOR ApplyUPProfile ON COMMAND IN (Air.CreateUPProfile, Air.UpdateUPProfile) AS Air.FillUPProfile;

    COMMAND Orders AS PbillFunc;
    COMMAND _Orders() AS PbillFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
    COMMAND Orders2(Untill.Orders) AS PbillFunc;
    COMMAND Orders3(Order Untill.Orders, Untill.PBill) AS PbillFunc;

    QUERY Query1 RETURNS QueryResellerInfoResult AS PbillFunc;
    QUERY _Query1() RETURNS Air.QueryResellerInfoResult AS PbillFunc WITH Comment=Air.PosComment, Tags=[Tag1, Air.Tag2];
    QUERY Query2(Untill.Orders) RETURNS QueryResellerInfoResult AS PbillFunc;
    QUERY Query3(Order Untill.Orders, Untill.PBill) RETURNS QueryResellerInfoResult AS PbillFunc;


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