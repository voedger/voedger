IMPORT SCHEMA 'server.com/account/repo/mypkg1';

TABLE MyTable INHERITS ODoc (
    myfield2 int32 NOT NULL,
    myfield3 mypkg1.mytype NOT NULL,
    myfield3 mypkg1.mytype2 NOT NULL
);
