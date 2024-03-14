WORKSPACE MyWorkspace1(
    TABLE MyTable1 INHERITS CDoc(
        Field1 varchar,
        Field2 varchar
    );
    TABLE MyTable11 INHERITS WDoc(
        Field11 varchar,
        Field22 bool
    );
-- 	TABLE NextNumbers INHERITS CSingleton (
-- 		NextPBillNumber int32
-- 	);
);
