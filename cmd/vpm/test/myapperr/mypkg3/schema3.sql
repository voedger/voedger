-- Copyright (c) 2023-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry' AS reg;

TABLE MyTable3 INHERITS ODoc (
    MyField int32 NOT NULL,
    Login ref(reg.Login) NOT NULL
);
