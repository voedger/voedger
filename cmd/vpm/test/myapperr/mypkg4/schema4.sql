-- Copyright (c) 2023-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'mypkg3';
IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

TABLE MyTable2 INHERITS ODoc (
    myfield2 int32 NOT NULL,
    myfield3 ref(mypkg3.MyTable1) NOT NULL,
    LoginHash ref(registry.Login) NOT NULL
);
