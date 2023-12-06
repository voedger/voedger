-- Copyright (c) 2021-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

TABLE MyTable1 INHERITS ODoc (
    MyField ref(registry.Login) NOT NUL
);
