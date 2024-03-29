# Command Processor
## Architecture
```mermaid
erDiagram
AppSchema ||--|{ TableSpec: declares
UserDefinedCmdFunctionSpec {
    string Name
    schema ArgumentObjectSchema
    schema ResultObjectSchema
    func Func
}
UserDefinedValidatorSpec {
    string Name
    schema ArgumentObjectSchema
    func Func
}
AppSchema ||--|{ UserDefinedCmdFunctionSpec: declares
AppSchema ||--|{ UserDefinedValidatorSpec: declares
UserDefinedCmdFunctionSpec ||--|| UserDefinedCmdFunction: "used to build"
UserDefinedValidatorSpec ||--|| UserDefinedValidator: "used to build"
UserDefinedValidator ||--|| Validator: is
TableSpec ||--|| CUDFunc: "used by"
CUDFunc ||--|| BuiltinCmdFunction: is
BuiltinCmdFunction }|--|| CommandFunction: is
UserDefinedCmdFunction ||--|| CommandFunction: is
CommandFunction ||--|| State: "reads from Storages using"
CommandFunction ||--|| Arg: "reads from"
CommandFunction ||--o{ Intents: "prepares"
CommandFunction ||--|| Result: "returns"
Result ||--|| CommandProcessor: "returned by"
Intents }o--|| CommandProcessor: "applied by"
Arg ||--|| CommandProcessor: "provided by"
State ||--|{ StateStorage: "reads from"
State ||--|| CommandProcessor: "provided by"
CommandFunction }|--|| CommandProcessor: "executed by"
Validator }|--|| CommandProcessor: "executed by"
CommandProcessor ||--|{ StateStorage: "applies intents to"
CommandProcessor ||--|{ ExtensionEngine: uses
ExtensionEngine }|--|| VVM: "provided by"
CommandProcessor }|--|| VVM: "created by"
CommandProcessor }|--|| Bus_HTTPProcessor: "called by"
Bus_HTTPProcessor ||--|| VVM: "created by"
```
## Notes
- Command Function can only generate IIntents of one type (CUDs).
- `cuds: [...]` is not a part of ANY command anymore
- `c.sys.CUD` is not available anymore as a separate function

## Rest API
- Every App's Table is represented by separate Rest API resource which internally executes sys.CUD command:
    - POST `/api/rest/<wsid>/<table_qname>`
        - batch operations supported: request is a Record OR array of Records;
        - record may include child records;
    - GET `/api/rest/<wsid>/<table_qname>/<id>`
        - a Record with it's child records is returned (ex. CDoc);
    - PATCH `/api/rest/<wsid>/<table_qname>/<id>`
        - record may include child records;
- Every App's Command represented by separate Rest API resource
    - POST `/api/rest/<wsid>/<cmd_qname>`


