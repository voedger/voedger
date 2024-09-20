Cleaned. We do not use it yet, also there is a problem. ref https://github.com/voedger/voedger/issues/740

It existed at:

- https://github.com/voedger/voedger/tree/73c8437bf07f7f2a7a87ea403ed79f08115ecf6e/pkg/ibus
- https://github.com/voedger/voedger/tree/73c8437bf07f7f2a7a87ea403ed79f08115ecf6e/pkg/ibusmem

### Bus

- Bus connects system services
  - E.g.: HTTP Server, EMail Gateway, App Partition Query Handler
- For CE/SE services are located in the same process

### Principles

- Limited number of concurrent requests: maxNumOfConcurrentRequests
  - Example: million of http connections but 1000 concurrent requests
  - "ibus.ErrBusUnavailable" (503) is returned if the number of concurrent requests is exceeded
- Sender and Receiver both respect timeouts: readWriteTimeout
  - E.g. 5 seconds, by (weak) analogy with [FoundationDB, Long-running read/write transactions](https://apple.github.io/foundationdb/anti-features.html)
- Result of QuerySender can be used even if AddressHandler has not been found - Sender will return `ErrReceiverNotFound` error

### Components

```mermaid
erDiagram
    ISender }|--|| Address : sends-to
    Address ||--|| AddressHandler : handled-by
    AddressHandler ||--|{ Processor : has
    Address ||--|| Receiver: associated-with

    Processor ||--|| Goroutine: has
    Processor }|--|| Receiver: has
    Goroutine }|--|| Receiver: calls
```

### How it works with Commands and Queries

```mermaid
erDiagram
    HTTPClient ||--|| ISender_1: "requested ISender for sys/registry/8/q"
    ISender_1 ||--|| Address_1: connected-to
    Address_1 ||--|| AddressHandler_1: served-by
    AddressHandler_1 ||--|| RegistryQueryHandler8: call

    HTTPClient ||--|| ISender_2: "requested ISender for sys/registry/5/c"
    ISender_2 ||--|| Address_2: connected-to
    Address_2 ||--|| AddressHandler_2: served-by
    AddressHandler_2 ||--|| RegistryCommandHandler5: call

    RegistryQueryHandler8 }o--|| Receiver: is
    RegistryCommandHandler5 }o--|| Receiver: is

    Address_1{
        string owner "sys"
        string app  "registry"
        int partition "8"
        int part    "q"
    }
    AddressHandler_1{
        goroutine processor1
        goroutine processor2
        goroutine processor3
        goroutine processor4
        goroutine processor5
    }
    Address_2{
        string owner "sys"
        string app  "registry"
        int partition "5"
        int part "c"
    }
    AddressHandler_2{
        goroutine r1
    }
```