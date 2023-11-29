IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

TABLE MyTable2 INHERITS ODoc (
    MyField ref(registry.NonexistentField) NT NULL
);
