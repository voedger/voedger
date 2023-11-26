# Extensions

## Extensions in Packages
```mermaid
erDiagram
Package ||--|{ SchemaFile: "has *.heeus"
Package ||..|| PackageSchema: defines
PackageSchema ||..|{ SchemaFile: "defined by"
PackageSchema ||--o{ ExtensionDef: has
Package ||--o{ ExtSrcFile: "has *.go, *.ts, *.lua"
Package ||--o| ExtBuildFile: "has build.sh"
ExtBuildFile ||..|{ ExtSrcFile: "compiles"
ExtBuildFile ||..|| ExtWasmFile: "produces pkg.wasm"
ExtensionDef ||..|{ ExtSrcFile: "implemented in"
ExtWasmFile ||..|| ExtensionModule: "is"
```
See also: 
- [Design: Extensions](https://github.com/heeus/heeus-design#extensions)

## Extension Engine Architecture
```mermaid
    erDiagram

    AppPartition ||--|{ CommandProcessorEngine: "has, one per EngineKind"
    AppPartition ||--|{ QueryProcessorEngine: "has, one per EngineKind"
    AppPartition ||--|{ Actualizer: has
    AppPartition ||--|| CommandProcessor: has
    AppPartition ||--|{ QueryProcessor: has

    CommandProcessorEngine ||..|| ExtensionEngine: "is"
    QueryProcessorEngine ||..|| ExtensionEngine: "is"
    Actualizer ||--|| ExtensionEngine: "has own"

    CommandProcessor ||..|| ExtensionsSite: "is"
    QueryProcessor ||..|| ExtensionsSite: "is"
    Actualizer ||..|| ExtensionsSite: "is"

    ExtensionsSite ||--|{ ExtensionPoint : "has"
    

    ExtensionPoint ||--|| ExtensionIO: "has"

    ExtensionEngine ||--|| ExtensionLimits : "has"
    ExtensionEngine ||..|| ExtensionEngineFactory : "created by"
    ExtensionEngine ||--|{ Memory: "has"
    ExtensionEngine ||--|{ Module: "instantiates"
    ExtensionPoint ||..|{ Extension: "invokes"
    ExtensionEngine ||..|{ Extension: "provides"
    
    Module ||--|{ Extension: "has"
    
    
    ExtensionLimits ||..|| ExecutionInterval: "can be"
    ExtensionLimits ||..|| IntentsLimit: "can be"


    ExtensionIO ||..|{ Extension: "used to read from State by"
    ExtensionIO ||..|{ Extension: "used to make Intents by"

    
    ExtEngineConfiguration ||--|| MemoryLimit: "includes"
    ExtEngineConfiguration ||--|| MemoryPoolSize: "includes"
    ExtensionEngineFactory ||..|| ExtEngineConfiguration: "uses"

    Memory ||..|| MemoryPoolSize: "as many as specified by"
```
See Also:
- Current Diagram [Extension Engines](https://github.com/heeus/heeus-design/#extension-engines)

## Common Principles: Extension Programming 
- Extensions are [pure functions](https://en.wikipedia.org/wiki/Pure_function). No global variables allowed.
- Any kind of input/output is done using State and Intents ONLY.
- (?) package name must be the same that is declared in PackageSchema


## BuildIn Extensions, Principles

`IBuiltInExtensionModule` with method `Register(name String, func BuiltinExtensionProc)` allowing register built-in functions:
```go
func provideSomeExtensions(cfg *istructsmem.AppConfigType) {
    // "MyCommand" must match the name of the command in VQL
    cfg.ExtCmdProc.Pkg("sys").BuiltinModule().Register("MyCommand", MyCommandImpl)
}
```
 
 All builtin functions has the same arguments:
```go
func MyCommandImpl(state istructs.IState, intents istructs.IIntents) {
    // ...
}
```

Rename storage `CmdResult` -> `Result`. It can be used in both Commands and Queries. Example for queries:
```go
// Some query function extension    
ext.ReadValues(viewPartialKey, func(key ext.TKey, value ext.TValue) {
    /*
        In QueryProcessor "Result" storage works differently:
        NewValue, created in ext.Result, will be sent through bus: 
            - every time a new value is created over the old one;
            - when the execution is over.
    */
    result := ext.NewValue(ext.KeyBuilder(ext.Result))  
    result.PutString("name", value.AsString("FullName"))
    result.PutInt32("age", customer.AsInt32("Age"))
})
```



