IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

TABLE MyTable3 INHERITS ODoc (
    MyField ref(registry.Login) NOT NULL
);
