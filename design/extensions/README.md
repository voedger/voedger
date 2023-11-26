## Architecture
```mermaid
erDiagram
Processor ||..|| IAppPartition: "borrows"
Processor ||..|| IEngine: "borrows"
IEngine ||--|{ Module: "instantiate, one per package"
IEngine ||--|| Invoke: "provides method"
Module ||--|{ Extension: "has"
Module ||--|| Memory: "has"

IAppPartition ||..|| IExtensionIO: "used to construct"
Invoke ||..|{ Extension: "invokes"
IExtensionIO ||..|| Invoke: "used by"
```

## IEngine interface
- Invoke(Name QName, Io ExtensionIO) (err error)

## Technical Issues
It seems `IAppPartition` and `IEngine` must be borrowed separately, because they have different lifetime in the Async Actualizer:
- `IExtensionIO` which is constructed from `IAppPartition` kept alive for multiple `Invoke` calls of the projector, until the Intents buffer is flushed.


```mermaid
sequenceDiagram
    participant IAppPartition
    participant IExtensionIO
    participant Actualizer
    participant IEngine
    IAppPartition-->>Actualizer: Borrow 
    activate IAppPartition
    Actualizer->>IExtensionIO: Construct
    
    IEngine-->>Actualizer: Borrow
    activate IEngine
    Actualizer->>IEngine: Invoke projector
    Actualizer-->>IEngine: Release
    deactivate IEngine

    IEngine-->>Actualizer: Borrow
    activate IEngine
    Actualizer->>IEngine: Invoke projector
    Actualizer-->>IEngine: Release
    deactivate IEngine

    Actualizer->>IExtensionIO: Flush

    Actualizer-->>IAppPartition: Release
    deactivate IAppPartition
```