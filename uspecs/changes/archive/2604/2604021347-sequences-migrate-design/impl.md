# Implementation plan: Sequences: migrate and actualize design

## Technical design

- [x] create: [apps/sequences--arch.md](../../../../specs/prod/apps/sequences--arch.md)
  - add: Migrate actual sequences design from [sequences-260109.md](sequences-260109.md) into Context Subsystem Architecture format
  - add: Goroutine interaction diagram showing how actualizer, flusher, and command processor goroutines interact via wait groups, channels, and contexts
  - add: Mark single seqID read per workspace stored in LRU cache as technical debt (TODO/FIXME): should always read all numbers per workspace and keep in memory without cache

- [x] create: [apps/sequences--arch2.md](../../../../specs/prod/apps/sequences--arch2.md)
  - add: Migrate proposed (complicated) sequences design from [sequences.md](sequences.md) into Context Subsystem Architecture format
