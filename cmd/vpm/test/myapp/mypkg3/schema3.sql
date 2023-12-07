-- Copyright (c) 2023-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

TABLE MyTable3 INHERITS ODoc (
    MyField ref(registry.Login) NOT NULL
);
