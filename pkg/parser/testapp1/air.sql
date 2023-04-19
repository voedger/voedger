SCHEMA air;

COMMENT BackofficeComment "Backoffice Comment";
COMMENT PosComment "Pos Comment";
TAG PosTag;
TAG BackofficeTag;

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
    TABLE ws_table OF ODOC (
        psname text,
        TABLE child (
            number int				
        )
    );	
);