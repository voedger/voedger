# Extension Engines
## Principles
- Extension engine is the component which invokes extensions
- Partition Processor has:
  - Extension engine instance(s)
  - Extension instance(s)
  - Host State (aka State 2.0)
## Overview
```mermaid
erDiagram
App ||--|{ AppPartition: has
AppPartition ||--|| CP: has
CP ||--|| HostState: has
CP ||--|{ ExtensionPackage: has
ExtensionPackage ||--|{ ExtensionInstance: has
ExtensionInstance {
    string name
    string extKind
}
ExtensionInstance }o--|| ExtEngine: invokes
ExtensionPackage ||--|{ ExtEngine: "has (one per engine kind)"
ExtEngine }|--|| HostState: uses
```

## Extension Engine Kinds
Principles:
- Heeus supports different extension engines (Builtin, WASM, LUA)
- Extension engine is constructed with the factory on partition processor initialization stage

```mermaid
erDiagram
ExtEngineFactory {
    int memLimit
}
ExtEngineFactory ||--|{ ExtEngine: creates
ExtEngine ||--|| WasmEngine: "can be"
ExtEngine ||--|| LuaEngine: "can be"
ExtEngine ||--|| BuiltinEngine: "can be"
WasmEngine ||--|| WasmRuntimeInstance: has
WasmEngine ||--|| WasmState: has
WasmEngine ||--|| ProxyWasm: has
WasmEngine ||--|| ClientMemory: has
WasmEngine ||--|{ UserFunctions: has
```


# WasmEngine

Principles:
- WasmEngine communicates with User Functions and WasmState through number of functions exported by WASM module (ProxyWasm)
- To communicate with HostState, WasmState uses number of functions implemented by Host and imported by WASM module (ProxyHost)

```mermaid
flowchart TD
    subgraph Host
    ProxyHost
    WasmEngine
    HostState
    ProxyHost-->HostState
    WasmEngine-->HostState
    end
    subgraph WASM
    UserFunctions-->WasmState
    ProxyWasm-->UserFunctions
    WasmState
    ProxyWasm
    end
    WasmEngine-->ProxyWasm
    WasmState-->ProxyHost
    ProxyWasm-->WasmState
```

# WasmEngine: Lifecycle
```mermaid
sequenceDiagram
    participant VVM
    participant CP
    participant ExtInstance
    participant WasmEngine
    participant ProxyWasm
    VVM->>ExtInstance: Create Instance
    VVM->>WasmEngine: Create Instance
    WasmEngine->>ProxyWasm: WasmAbiVersion()
    WasmEngine->>ProxyWasm: WasmInit()
    VVM->>CP: Create Instance (ExtInstances, WasmEngine)
    loop commands
    CP->>ExtInstance: Invoke(args)
    ExtInstance->>WasmEngine: Invoke(name, args)
    WasmEngine->>ProxyWasm: WasmRun()
    WasmEngine->>ProxyWasm: WasmReset()
    end
    VVM->>WasmEngine: Finit()
    WasmEngine->>ProxyWasm: WasmFinit()
```

# ProxyWasm ABI: Detailed
```mermaid
sequenceDiagram
    participant ExtInstance
    participant WasmEngine
    participant HostState
    participant ProxyWasm
    participant ProxyHost
    participant WasmState
    participant WasmExt
    ExtInstance->>+WasmEngine: Invoke(name, args)
    WasmEngine->>+ProxyWasm: WasmRun(sz_name, kind, arg_ptr)
    ProxyWasm-->>WasmState: Reset
    ProxyWasm->>WasmExt: UserFunction()
    activate WasmExt
    WasmExt->>WasmState: KeyBuilder(storage, ...)
    WasmState->>WasmExt: WasmKeyBuilder
    note right of WasmExt: fill key
    WasmExt->>+WasmState: MustExist(key)
    WasmState->>+ProxyHost: HostGetElem(key_ptr)
    ProxyHost->>HostState: KeyBuilder(storage, ...)
    ProxyHost->>HostState: MustExist(key)
    HostState->>ProxyHost: IStateElement
    ProxyHost->>-WasmState: elem_ptr
    WasmState->>-WasmExt: WasmStateElement
    WasmExt->>WasmState: ValueBuilder(key)
    WasmState->>WasmExt: WasmElementBuilder
    note right of WasmExt: fill element
    WasmExt->>ProxyWasm: Done Run()
    deactivate WasmExt
    ProxyWasm->>-WasmEngine: Done WasmRun *intents
    WasmEngine->>HostState: NewValue(key1)
    WasmEngine->>HostState: NewValue(key2)
    WasmEngine->>+ProxyWasm: WasmReset()
    ProxyWasm->>+WasmState: Reset()
    ProxyWasm->>-WasmEngine: Done WasmReset()
    WasmEngine->>-ExtInstance: Done Invoke
    ExtInstance->>HostState: Apply

```

# ProxyWasm ABI: Read Detailed
```mermaid
sequenceDiagram
    participant HostState
    participant ProxyWasmHost
    participant ProxyWasm
    participant WasmState
    participant WasmExt
    WasmExt->>WasmState: KeyBuilder(storage, ...)
    WasmState->>WasmExt: WasmKeyBuilder
    note right of WasmExt: fill key
    WasmExt->>+WasmState: Read(ctx_id, key)
    WasmState->>+ProxyWasmHost: HostReadElems(ctx_id, key_ptr)
    ProxyWasmHost->>HostState: KeyBuilder(storage, ...)
    ProxyWasmHost->>HostState: Read(key, callback)
    HostState->>ProxyWasmHost: IStateElement
    ProxyWasmHost->>ProxyWasm: WasmOnRead(ctx_id, key_ptr, elem_ptr)
    ProxyWasm->>WasmExt: OnStateRead(ctx_id, key, value)
    HostState->>ProxyWasmHost: IStateElement
    ProxyWasmHost->>-ProxyWasm: WasmOnRead(ctx_id, key_ptr, elem_ptr)
    ProxyWasm->>WasmExt: OnStateRead(ctx_id, key, value)
    WasmState->>-WasmExt: Done Read

```



# Wasm ABI
The list of functions to be exported by WASM:
```go
    WasmAbiVersion_X_Y()

    // Called by host when client is initialized
    WasmInit()

    // Called by host to run the extension
    // sz_name and arg_ptr must be released by host when ClientRun is finished
    // ext_kind: CmdFunction, QueryFunction, Validator, Projector, Etc
    WasmRun(sz_name int32, ext_kind int32 arg_ptr int32) (result int32)

    // Called by host to allocate memory on client
    WasmMalloc(size int32) (ptr int32)

    // Called by host after Run
    WasmReset()

    // Called by host when client is finalized
    // Must cleanup the resources
    WasmFinit()

    // Called by HostReadElems
    WasmOnRead(ctx_id int32, key_ptr int32, element_ptr int32)
```

The list of functions to be imported by WASM:
```go
    // Gets element.
    // Host calls ClientMalloc to allocate memory in WASM VM
    // Host returns elem_ptr which must be released by client
    // Returns:
    //   0 - ok, element exists
    //   1 - ok, element not exists
    //   2 - key validation issue
    //   3 - i/o error
    //   4 - memory allocation issue
    HostGetElem(key_ptr int32, elem_ptr* int32) (result int32)


    // Read elements and calls ClientOnRead for every element read
    // Returns:
    //   0 - ok
    //   2 - key validation issue
    //   3 - i/o error
    HostReadElems(ctx_id int32, key_ptr int32) (result int32)
```


# References
- [WebAssembly объединит их всех](https://habr.com/ru/post/671048/)
