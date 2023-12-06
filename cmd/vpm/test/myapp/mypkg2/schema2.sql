-- Copyright (c) 2021-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'server.com/account/repo/mypkg1';

TABLE MyTable INHERITS ODoc (
    myfield2 int32 NOT NULL,
    myfield3 ref(mypkg1.MyTable1) NOT NULL
);
