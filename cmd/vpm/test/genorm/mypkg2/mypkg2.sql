IMPORT SCHEMA 'mypkg1';

WORKSPACE MyWorkspace2(
    TABLE MyTable2 INHERITS ODoc(
        Field3 varchar,
        Field4 int32--,
        --Field1 ref(mypkg1.MyWorkspace1.MyTable1) NOT NULL
    );
);
