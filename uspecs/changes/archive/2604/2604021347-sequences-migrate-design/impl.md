# Implementation plan: Sequences: migrate and actualize design

## Technical design

- [x] create: [apps/sequences--arch.md](../../../../specs/prod/apps/sequences--arch.md)
  - add: Migrate actual sequences design into Context Subsystem Architecture format
  - add: Key components with deep code links (command processor components, ISequencer package, storage implementations)
  - add: Data structures (appPartition, workspace, implIIDGenerator) with key constants
  - add: Key flows (initialization, recovery, command processing steps)
  - add: Sequencing transaction diagram (target flow for ISequencer)
  - add: Goroutine lifecycle and interaction diagram
  - add: Synchronization primitives documentation
  - add: Mark single seqID read per workspace stored in LRU cache as technical debt (FIXME)
  - fix: Separate active command processor behavior from ISequencer package (implemented, not yet integrated)

- [x] create: [apps/sequences--arch2.md](../../../../specs/prod/apps/sequences--arch2.md)
  - add: Migrate proposed (complicated) sequences design into Context Subsystem Architecture format
  - fix: Correct "Most Recently Used (LRU)" to "Least Recently Used (LRU)"

## Source cleanup

- [x] Remove `reqmd` tags from 13 Go source files
  - pkg/appparts/internal/seqstorage (provide.go, type.go)
  - pkg/isequencer (impl.go, interface.go, isequencer_test.go, types.go)
  - pkg/istructsmem (impl.go)
  - pkg/vit (impl.go)
  - pkg/vvm (impl_cfg.go, types.go)
  - pkg/vvm/storage (consts.go, consts_test.go, impl_seqstorage.go)
