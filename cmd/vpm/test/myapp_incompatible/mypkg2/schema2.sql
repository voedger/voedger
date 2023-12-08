-- Copyright (c) 2023-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'server.com/account/repo/mypkg1';

TABLE MyTable INHERITS ODoc (
    myfield3 ref(mypkg1.MyTable1) NOT NULL, -- incompatibility: OrderChanged
    myfield2 int32 NOT NULL -- incompatibility: OrderChanged
);
