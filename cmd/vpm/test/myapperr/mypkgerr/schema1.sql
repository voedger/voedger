IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

TABLE MyTable1 INHERITS ODocUnknown (
    MyField ref(registry.NonexistentField) NT NULL
);
