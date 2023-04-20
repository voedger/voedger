SCHEMA air;

IMPORT SCHEMA "github.com/untillpro/untill";
IMPORT SCHEMA "github.com/untillpro/airsbp" AS air;		

COMMENT BackofficeComment "Backoffice Comment";
TAG BackofficeTag;

FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
FUNCTION MyTableValidator(TableRow) RETURNS string ENGINE WASM;
FUNCTION MyTableValidator(param1 aaa.TableRow, bbb.string) RETURNS ccc.TableRow ENGINE WASM;

ROLE UntillPaymentsUser;

TABLE air_table_plan OF CDOC (
    fstate int,
    name text NOT NULL,
    vf text NOT NULL VERIFIABLE,
    i1 int DEFAULT 1,
    s1 text DEFAULT "a",
    ii int DEFAULT NEXTVAL(sequence),
    id_bill int64 REFERENCES air.bill,
    ckf text CHECK "^[0-9]{8}$",
    UNIQUE fstate, name
) WITH Comment=BackofficeComment, Tags=[BackofficeTag];

WORKSPACE ws (

    COMMENT PosComment "Pos Comment";
    TAG PosTag;

	USE TABLE somepackage.sometable;
	USE TABLE mytable;
	USE TABLE untill.*; 

    ROLE LocationManager;

    PROJECTOR ON COMMAND air.CreateUPProfile AS SomeFunc;
    PROJECTOR ON COMMAND ARGUMENT untill.QNameOrders AS xyz.SomeFunc2;
    PROJECTOR ON INSERT untill.bill AS SomeFunc;
    PROJECTOR ON INSERT OR UPDATE untill.bill AS SomeFunc;
    PROJECTOR ON UPDATE untill.bill AS SomeFunc;
    PROJECTOR ON UPDATE OR INSERT untill.bill AS SomeFunc;
    PROJECTOR ApplyUPProfile ON COMMAND IN (air.CreateUPProfile, air.UpdateUPProfile) AS air.FillUPProfile;

    COMMAND Orders AS PbillFunc;
    COMMAND Orders() AS PbillFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
    COMMAND Orders2(untill.orders) AS PbillFunc;
    COMMAND Orders3(order untill.orders, untill.pbill) AS PbillFunc;

    QUERY Query1 RETURNS QueryResellerInfoResult AS PbillFunc;
    QUERY Query1() RETURNS air.QueryResellerInfoResult AS PbillFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
    QUERY Query2(untill.orders) RETURNS QueryResellerInfoResult AS PbillFunc;
    QUERY Query3(order untill.orders, untill.pbill) RETURNS QueryResellerInfoResult AS PbillFunc;


    GRANT ALL ON ALL TABLES WITH TAG untill.Backoffice TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE untill.orders TO LocationUser;
    GRANT EXECUTE ON COMMAND Orders TO LocationUser;
    GRANT EXECUTE ON QUERY TransactionHistory TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    TABLE ws_table OF CDOC, air.SomeType (
        psname text,
        TABLE child (
            number int				
        )
    );	

    VIEW XZReports(
        Year int32,
        Month int32, 
        Day int32, 
        Kind int32, 
        Number int32, 
        XZReportWDocID id
    ) AS RESULT OF air.UpdateXZReportsView
    WITH 
        PrimaryKey="(Year), Month, Day, Kind, Number",
        Comment=PosComment;


    RATE BackofficeFuncRate1 1000 PER HOUR;
    RATE BackofficeFuncRate2 100 PER MINUTE PER IP;

);