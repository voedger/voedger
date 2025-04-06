# Package isequencer

## Overview
The project implements a scalable, monotonic sequence generator for various sequence IDs in a distributed environment. It ensures ordered number generation, efficient caching, and reliability through background flushing and actualization.

## Key Components

- **server/design/sequences.md**<br/> 
Describes high-level design goals, motivation, and use cases for sequence management in the platform.
Outlines initial sequence values and the role of caching and projections.


- **pkg/isequencer/design.md**<br/>
Explains the workflow, including how the sequencer is started, how actualizer and flusher goroutines work, and how sequences are persisted into storage.


- **pkg/isequencer/impl.go**<br/>
Contains the core sequencer methods:
  - `Start(wsKind, wsID)` begins a sequencing transaction and returns the next offset.
  - `Next(seqID)` obtains new sequence values.
  - `Flush()` commits in-memory values to be flushed.
  - `Actualize()` cancels the current transaction and synchronizes sequences with the persisted state.

- **pkg/isequencer/types.go**<br/>
  Defines common data types (`SeqID`, `WSID`, `PLogOffset`, etc.) and includes the parameter configuration struct for the sequencer’s caching, batching, and storage limits.


## Features
1. **Monotonic Number Generation**<br/>
Ensures strictly increasing sequence values for operations like record IDs and log offsets.

2. **Caching & LRU**<br/>
Frequently accessed sequences are stored in an LRU cache, reducing persistent lookups.


3. **Background Flush & Actualization**<br/>

- `Flush` writes cached numbers to durable storage.
- `Actualize` reconciles in-memory states with storage, enabling error recovery.

4. **Batch Processing**<br/>
Accumulated sequence updates are written in batches to minimize performance overhead.


5. **Retry Mechanisms**<br/>
Robust retries handle transient storage issues during flush and actualization phases.


## Usage
1. **Initialization**: Each partition instantiates an `isequencer.ISequencer` with configured `SeqTypes`.
2. **Start Event**: Call `Start(wsKind, wsID)` to begin a transaction and obtain the current offset.
3. **Next Value**: Use `Next(seqID)` to retrieve or allocate new sequence numbers.
4. **Flush or Actualize**:
- Use `Flush()` after successful operations to finalize in-memory increments.
- Use `Actualize()` on errors to discard unflushed transitions and realign with persisted data.

## Testing
The tests cover scenarios like handling invalid calls, recovery, high concurrency, permanent storage failures, and ensuring monotonic growth under different sequences.