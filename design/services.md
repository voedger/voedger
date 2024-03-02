## Top-level sections

- [General principles: WASM](#general-principles-wasm)
- [General principles: VSQL DDL](#general-principles-vsql-ddl)
- [Development](#development)

---
## General principles: WASM

Development

- Extension language
  - Development: Go
  - Compilation: TinyGo

Runtime
  - wazero
  - Preallocated Buffer
    - Used to send data from the host
    - WasmPreallocatedBufferSize: 64K + 25% growth if necessary
    - Per wazero module
  - Garbage collection is not used
    - Reason: too slow (even when runtime is a compiler) and unreliable
    - In case of memory overflow the memory is restored to the initial state and function call is repeated

### Related components

Voedger Engine
- [pkg: iextengine](../../pkg/iextengine)
- [pkg: iextenginebuiltin](../../pkg/iextenginebuiltin)
- [pkg: iextenginewazero](../../pkg/iextenginewazero/README.md)

SDK
- [pkg: github.com/voedger/exttinygo](../../staging/src/github.com/voedger/exttinygo/README.md)    

---
## General principles: VSQL DDL

- No NULL values
  - NULL-reference is just zero value
- Not possible to update an unique field

### Related components
- [pkg: parser](../../pkg/parser)

---
## Development
### Principles

*Folder Structure*
- vddl files
- `extwasm` folder

### Related components

- [cli: vpm](../../cmd/vpm/README.md)


