# Application Partitions

Application partitions are a way to split up the application into smaller storage and processing units. This is useful for large applications that need to be split up into smaller parts to make it easier to deploy and update.

## Components

```mermaid
erDiagram
    apps ||--|{ appRT : "manages"
    appRT ||--|{ appPartitionRT : "contains"
    appRT ||--|| appVersion : "has latest"
    appVersion ||--|{ pool : "has"
    appPartitionRT ||--|| syncActualizer : "has"
    appPartitionRT ||--|| PartitionActualizers : "has"
    appPartitionRT ||--|| PartitionSchedulers : "has"
    appPartitionRT ||--|| Limiter : "has"
    appPartitionRT ||--o{ borrowedPartition : "lends"
    borrowedPartition ||--|| engines : "borrows"
    pool }|--|| engines : "manages"

    apps {
        sync_RWMutex mx
        context vvmCtx
        map[AppQName]appRT apps
    }
    appRT {
        sync_RWMutex mx
        AppQName name
        NumAppPartitions partsCount
        map[PartitionID]appPartitionRT parts
    }
    appVersion {
        sync_RWMutex mx
        IAppDef def
        IAppStructs structs
        Pool[engines] pools[4]
    }
    appPartitionRT {
        AppQName app
        PartitionID id
        ISyncOperator syncActualizer
    }
    borrowedPartition {
        ProcessorKind kind
        engines engines
    }
    engines {
        m _ "map[ExtensionEngineKind]IExtensionEngine"
    }
```