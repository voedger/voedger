## Principles

### WASM  

Development

- Extension language
  - Development: Go
  - Compilation: TinyGo
- Extension folder structure
  - `extwasm` folder

Runtime
  - wazero
  - Preallocated Buffer
    - Used to send data from the host
    - WasmPreallocatedBufferSize: 64K + 25% growth if necessary
    - Per wazero module
  - Garbage collection is not used
    - Too slow (even when runtime is a compiler) and unreliable
    - In case of memory overflow the memory is restored to the initial state    

## Components

- [cli: vpm](../../cmd/vpm/README.md)
- [pkg: iextengine](../../pkg/iextengine)
- [pkg: iextenginebuiltin](../../pkg/iextenginebuiltin)
- [pkg: iextenginewazero](../../pkg/iextenginewazero/README.md)
- [pkg: parser](../../pkg/parser)
- [pkg: exttinygo](../../staging/src/github.com/voedger/exttinygo/README.md)
