# Context subsystem architecture: Sequences (proposed design)

> **Status**: Proposed — not implemented. Deemed too complicated; kept for reference

## Motivation

As of March 1, 2025, the sequence implementation has several critical limitations that impact system performance and scalability:

- **Unbound Memory Growth**: Sequence data for all workspaces is loaded into memory simultaneously, creating a direct correlation between memory usage and the number of workspaces. This approach becomes unsustainable as applications scale.

- **Prolonged Startup Times**: During command processor initialization, a resource-intensive "recovery process" must read and process the entire PLog to determine the last used sequence numbers. This causes significant startup delays that worsen as event volume grows.

The proposed redesign addresses these issues through intelligent caching, background updates, and optimized storage mechanisms that maintain sequence integrity while dramatically improving resource utilization and responsiveness.

## Introduction

This document outlines the design for sequence number management within the Voedger platform.

A **Sequence** in Voedger is defined as a monotonically increasing series of numbers. The platform provides a unified mechanism for sequence generation that ensures reliable, ordered number production.

As of March 1, 2025, Voedger implements four specific sequence types using this mechanism:

- **PLogOffsetSequence**: Tracks write positions in the PLog
  - Starts from 1
- **WLogOffsetSequence**: Manages offsets in the WLog
  - Starts from 1
  - To read all events by SELECT
- **CRecordIDSequence**: Generates unique identifiers for CRecords
  - Starts from 322685000131072
    - [https://github.com/voedger/voedger/blob/ae787d43bf313f8e3119b8b2ce73ea43969eaaa3/pkg/istructs/utils.go#L35](https://github.com/voedger/voedger/blob/ae787d43bf313f8e3119b8b2ce73ea43969eaaa3/pkg/istructs/utils.go#L35)
  - Motivation:
    - Efficient CRecord caching on the DBMS side (Most CRecords reside in the same partition)
    - Simple iteration over CRecords
- **OWRecordIDSequence**: Provides sequential IDs for ORecords/WRecords (OWRecords)
  - Starts from 322680000131072
  - There are a potentially lot of such records, so it is not possible to use SELECT to read all of them

As the Voedger platform evolves, the number of sequence types is expected to expand. Future development will enable applications to define their own custom sequence types, extending the platform's flexibility to meet diverse business requirements beyond the initially implemented system sequences.

These sequences ensure consistent ordering of operations, proper transaction management, and unique identification across the platform's distributed architecture. The design prioritizes performance and scalability by implementing an efficient caching strategy and background updates that minimize memory usage and recovery time.

### Background

- [#688: record ID leads to different tables](https://github.com/voedger/voedger/issues/688)
- [VIEW RecordsRegistry](https://github.com/voedger/voedger/blob/ec85a5fed968e455eb98983cd12a0163effdc096/pkg/sys/sys.vsql#L260)
- [Singleton IDs](https://github.com/voedger/voedger/blob/ec85a5fed968e455eb98983cd12a0163effdc096/pkg/istructs/consts.go#L101)

Existing design

- [const MinReservedBaseRecordID = MaxRawRecordID + 1](https://github.com/voedger/voedger/blob/84babc48d63107e29fcec28eefb0a461b5a34474/pkg/istructs/consts.go#L96)
  - 65535 + 1
  - const MaxReservedBaseRecordID = MinReservedBaseRecordID + 0xffff // 131071
  - const FirstSingletonID = MinReservedBaseRecordID // 65538
  - const MaxSingletonID = MaxReservedBaseRecordID // 66047, 512 singletons
- [ClusterAsRegisterID = 0xFFFF - 1000 + iota](https://github.com/voedger/voedger/blob/84babc48d63107e29fcec28eefb0a461b5a34474/pkg/istructs/consts.go#L69)
  - ClusterAsCRecordRegisterID
- [const FirstSingletonID](https://github.com/voedger/voedger/blob/84babc48d63107e29fcec28eefb0a461b5a34474/pkg/istructs/consts.go#L101)
- [cmdProc.appsPartitions](https://github.com/voedger/voedger/blob/b7fa6fa9e260eac4f1de80312c14ad4250f400a3/pkg/processors/command/provide.go#L32)
- [command/impl.go/getIDGenerator](https://github.com/voedger/voedger/blob/b7fa6fa9e260eac4f1de80312c14ad4250f400a3/pkg/processors/command/impl.go#L299)
- [command/impl.go: Put IDs to response](https://github.com/voedger/voedger/blob/b7fa6fa9e260eac4f1de80312c14ad4250f400a3/pkg/processors/command/impl.go#L790)
- [command/impl.go: (idGen \*implIDGenerator) NextID](https://github.com/voedger/voedger/blob/b7fa6fa9e260eac4f1de80312c14ad4250f400a3/pkg/processors/command/impl.go#L816)
- [istructmem/idgenerator.go: (g \*implIIDGenerator) NextID](https://github.com/voedger/voedger/blob/b7fa6fa9e260eac4f1de80312c14ad4250f400a3/pkg/istructsmem/idgenerator.go#L32)
  - [onNewID](https://github.com/voedger/voedger/blob/b7fa6fa9e260eac4f1de80312c14ad4250f400a3/pkg/istructsmem/idgenerator.go#L16)

#### Previous flow

- Recovery on the first request into the workspace
  - CP creates new `istructs.IIDGenerator` [instance](https://github.com/voedger/voedger/blob/9d400d394607ef24012dead0d59d5b02e2766f7d/pkg/processors/command/impl.go#L136)
  - the `istructs.IIDGenerator` instance is kept for the WSID
  - `istructs.IIDGenerator` instance is tuned with the data from the each event of the PLog:
    - for each CUD:
      - CUD.ID is set as the current RecordID
        - `IIDGenerator.UpdateOnSync` [is called](https://github.com/voedger/voedger/blob/9d400d394607ef24012dead0d59d5b02e2766f7d/pkg/processors/command/impl.go#L253)
- save the event after cmd exec:
  - `istructs.IIDGenerator` [instance is provided to `IEvents.PutPlog()`](https://github.com/voedger/voedger/blob/9d400d394607ef24012dead0d59d5b02e2766f7d/pkg/processors/command/impl.go#L307)
  - `istructs.IIDGenerator.Next()` is called to convert rawID->realID for ODoc in arguments and [each resulting CUD](https://github.com/voedger/voedger/blob/9d400d394607ef24012dead0d59d5b02e2766f7d/pkg/istructsmem/event-types.go#L189)

#### Actual Sequences design as of 26-01-09

- [sequences-260109.md](sequences-260109.md)

---

## Definitions

**APs**: Application Partitions

**SequencesTrustLevel**:

The `SequencesTrustLevel` setting determines how events and table records are written.

| Level | Events, write mode | Table Records, write mode |
| ----- | ------------------ | ------------------------- |
| 0     | InsertIfNotExists  | InsertIfNotExists         |
| 1     | InsertIfNotExists  | Put                       |
| 2     | Put                | Put                       |

**Note**
`SequencesTrustLevel` is not used for the case when we're calling `PutPlog()` to mark the event as corrupted. `Put()` always used in this case

## Analysis

### Sequencing strategies

As of March 1, 2025, record ID sequences may overlap, and only 5,000,000,000 IDs are available for OWRecords, since OWRecord IDs start from 322680000131072, while CRecord IDs start from 322685000131072.

Solutions:

- **One sequence for all records**:
  - Pros:
    - 👍Clean for Voedger users
    - 👍IDs are more human-readable
    - 👍Simpler Command Processor
  - ❌Cons: CRecords are not cached efficiently
    - Solution: Let the State read copies of CRecords from sys.Collection, or possibly from an alternative optimized storage to handle large CRecord data
      - ❌Cons: Why we need CRecords then
      - 👍Pros: Separation of write and read models
- **Keep as is**:
  - Pros
    - 👍Easy to implement
  - Cons
    - ❌ No separation between write and read models
    - ❌ Only 5 billions of OWRecords (ClusterAsRegisterID < ClusterAsCRecordRegisterID)
      - Solution: Configure sequencer to use multiple ranges to avoid collisions
        - 👍Pros: Better control over sequences

### SequencesTrustLevel: Performance impact

- https://snapshots.raintank.io/dashboard/snapshot/zEW5AQHECtKLIcUeO2PJnmy3nkQDhp9m?orgId=0
  - Zero SequencesTrustLevel was introduced to the Air performance testbench on 2025-04-29
  - Latency is increased from 40 ms to 120 ms with spikes up to 160 ms
  - Testbench throughput reduced from 4000 command per seconds to 1400 cps
  - CPU usage is decreased from 75% to 42%
  - So we can make an educated guess that maximum throughput would be reduced by 4000 / 1400 \* 42 / 75 = 1.6 times

## Solution overview

The proposed approach implements a more efficient and scalable sequence management system through the following principles:

- **Projection-Based Storage**: Each application partition will maintain sequence data in a dedicated projection ???(`SeqData`). SeqData is a map that eliminates the need to load all sequence data into memory at once
- **Offset Tracking**: `SeqData` will include a `SeqDataOffset` attribute that indicates the PLog partition offset for which the stored sequence data is valid, enabling precise recovery and synchronization
- **LRU Cache Implementation**: Sequence data will be accessed through a Least Recently Used (LRU) cache that keeps frequently accessed sequences in memory while evicting the least recently accessed ones
- **Background Updates**: As new events are written to the PLog, sequence data will be updated in the background, ensuring that the system maintains current sequence values without blocking operations
- **Batched Writes**: Sequence updates will be collected and written in batches to reduce I/O operations and improve throughput
- **Optimized Actualization**: The actualization process will use the stored `SeqDataOffset` to process only events since the last known valid state, dramatically reducing startup times

This approach decouples memory usage from the total number of workspaces and transforms the recovery process from a linear operation dependent on total event count to one that only needs to process recent events since the last checkpoint.

## Use cases

### VVMHost: Configure SequencesTrustLevel mode for VVM

VVMHost uses VVMConfig.SequencesTrustLevel.

### CP: Handling SequencesTrustLevel for Events

- SequencesTrustLevelForPLog
  - When PLog is written then SequencesTrustLevel is used to determine the write mode
  - Note: except the `update corrupted` case
- SequencesTrustLevelForWLog
  - When WLog is written then SequencesTrustLevel is used to determine the write mode
  - Note: except the case when the wlog event was already stored before. Consider PutWLog is called to re-apply the last event

### CP: Handling SequencesTrustLevel for Table Records

- SequencesTrustLevelForRecords
  - When a record is inserted SequencesTrustLevel is used to determine the write mode
  - When a record is updated - nothing is done in connection with SequencesTrustLevel

### CP: Command processing

- StartSequencesGeneration
  - When: CP starts processing a request
  - Flow:
    - `partitionID` is calculated using request WSID and amount of partitions declared [in AppDeploymentDescriptor here](https://github.com/voedger/voedger/blob/9d400d394607ef24012dead0d59d5b02e2766f7d/pkg/vvm/impl_requesthandler.go#L61)
    - sequencer, err := IAppPartition.Sequencer() err
    - nextPLogOffest, ok, err := sequencer.Start(wsKind, WSID)
      - if !ok
        - Actualization is in progress
        - Flushing queue is full
        - Returns 503: "server is busy"
- NextSequenceNumber
  - When: After CP starts sequences generation
  - Flow:
    - sequencer.Next(sequenceId)
- FlushSequenceNumbers
  - When: After CP saves the PLog record successfully
  - Flow:
    - sequencer.Flush()
- ReactualizeSequences
  - When: After CP fails to save the PLog record
  - Flow:
    - sequencer.Actualize()

### IAppPartitions implementation: Instantiate sequencer on Application deployment

- When: Partition with the `partitionID` is deployed
- Flow:
  - Instantiate the implementation of the `isequencer.ISeqStorage` (appparts.internal.seqStorage, see below)
  - Instantiate `sequencer := isequencer.New(*isequencer.Params)`
  - Save `sequencer` so that it will be returned by IAppPartition.Sequencer()

---

## Technical design: Components

### IAppPartition.Sequencer

- Description: Returns `isequencer.ISequencer`
- Covers: StartSequencesGeneration

### VVMConfig.SequencesTrustLevel

- Covers: VVMConfig.ConfigureSequencesTrustLevel

---

### pkg/isequencer

Core:

- ISequencer: Interface for working with sequences
- sequencer: Implementation of the `isequencer.ISequencer` interface
  - Start: Starts Sequencing Transaction for the given WSID
  - Next: Returns the next sequence number for the given SeqID
  - Flush: Completes Sequencing Transaction
  - Actualize: Cancels Sequencing Transaction and starts the Actualization process

Tests:

- mockISeqStorage: Mock implementation of `isequencer.ISeqStorage` for testing purposes
- NewMustStartActualization: `isequencer.New()` must start the Actualization process, Start() must return `0, false`
  - Design: blocking hook in mockISeqStorage
- Race: If !t.Short() run something like `go test ./... -count 50 -race`

Some edge case tests:

- MultipleActualizes
  - Params.MaxNumUnflushedValues = 5
  - For numOfEvents in [0, 10*Params.MaxNumUnflushedValues]
    - Create a new ISequencer instance
    - Check that Next() returns correct values after recovery
- LongRecovery
  - Repeat { Start {Next} randomly( Flush | Actualize ) } cycle 100 times
  - Check that the system recovers well
  - Check that the sequence values are increased monotonically
- FlushPermanentlyFails
  - Recovery has worked but then ISeqStorage.WriteValuesAndOffset() fails permanently
  - First Start/Flush must be ok
  - Some of the next Start must not be ok
  - Flow:
    - MaxNumUnflushedValues = 5
    - Recover
    - Mock error on WriteValuesAndOffset
    - Start/Next/Flush must be ok
    - loop Start/Next/Flush until Start() is not ok (the 6th times till unflushed values exceed the limit)

#### interface.go

```go
/*
 * Copyright (c) 2025-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package isequencer

import (
  "context"
  "time"
)

type SeqID      uint16
type WSKind     uint16
type WSID       uint64
type Number     uint64
type PLogOffset uint64

type NumberKey struct {
  WSID  WSID
  SeqID SeqID
}

type SeqValue struct {
  Key    NumberKey
  Value  Number
}

// To be injected into the ISequencer implementation.
//
type ISeqStorage interface {

  // If number is not found, returns 0
  ReadNumbers(WSID, []SeqID) ([]Number, error)
  ReadNextPLogOffset() (PLogOffset, error)

  // IDs in batch.Values are unique
  // len(batch) may be 0
  // offset: Next offset to be used
  // batch MUST be written first, then offset
  WriteValuesAndNextPLogOffset(batch []SeqValue, nextPLogOffset PLogOffset) error

  // ActualizeSequencesFromPLog scans PLog from the given offset and send values to the batcher.
  // Values are sent per event, unordered, ISeqValue.Keys are not unique.
  // err: ctx.Err() if ctx is closed
  ActualizeSequencesFromPLog(ctx context.Context, offset PLogOffset, batcher func(batch []SeqValue, offset PLogOffset) error) (err error)
}

// ISequencer defines the interface for working with sequences.
// ISequencer methods must not be called concurrently.
// Use: { Start {Next} ( Flush | Actualize ) }
//
// Definitions
// - Sequencing Transaction: Start -> Next -> (Flush | Actualize)
// - Actualization: Making the persistent state of the sequences consistent with the PLog.
// - Flushing: Writing the accumulated sequence values to the storage.
// - LRU: Least Recently Used cache that keep the most recent next sequence values in memory.
type ISequencer interface {

  // Start starts Sequencing Transaction for the given WSID.
  // Marks Sequencing Transaction as in progress.
  // Panics if Sequencing Transaction is already started.
  // Normally returns the next PLogOffset, true
  // Returns `0, false` if:
  // - Actualization is in progress
  // - The number of unflushed values exceeds the maximum threshold
  // If ok is true, the caller must call Flush() or Actualize() to complete the Sequencing Transaction.
  Start(wsKind WSKind, wsID WSID) (plogOffset PLogOffset, ok bool)

  // Next returns the next sequence number for the given SeqID.
  // Panics if Sequencing Transaction is not in progress.
  // err: ErrUnknownSeqID if the sequence is not defined in Params.SeqTypes.
  Next(seqID SeqID) (num Number, err error)

  // Flush completes Sequencing Transaction.
  // Panics if Sequencing Transaction is not in progress.
  Flush()

  // Actualize cancels Sequencing Transaction and starts the Actualization process.
  // Panics if Actualization is already in progress.
    // Flow:
  // - Mark Sequencing Transaction as not in progress
  // - Do Actualization process
  //   - Cancel and wait Flushing
  //   - Empty LRU
  //   - Write next PLogOffset
  Actualize()

}

// Params for the ISequencer implementation.
type Params struct {

  // Sequences and their initial values.
  // Only these sequences are managed by the sequencer (ref. ErrUnknownSeqID).
  SeqTypes map[WSKind]map[SeqID]Number

  SeqStorage ISeqStorage

  MaxNumUnflushedValues int           // 500
  // Size of the LRU cache, NumberKey -> Number.
  LRUCacheSize int // 100_000

  BatcherDelay time.Duration // 5 * time.Millisecond
}
```

#### Implementation requirements

```go
// filepath: pkg/isequencer: impl.go

import (
  "context"
  "time"

  "github.com/hashicorp/golang-lru/v2"
)

// Implements isequencer.ISequencer
// Keeps next (not current) values in LRU and type ISeqStorage interface
type sequencer struct {
  params *Params

  actualizerInProgress atomic.Bool

  // Set by s.Actualize(), never cleared (zeroed).
  // Used by s.cleanup().
  actualizerCtxCancel context.CancelFunc
  actualizerWG  *sync.WaitGroup


  // Cleared by s.Actualize()
  lru *lru.Cache

  // Initialized by Start()
  // Example:
  // - 4 is the offset ofthe last event in the PLog
  // - nextOffset keeps 5
  // - Start() returns 5 and increments nextOffset to 6
  nextOffset PLogOffset

  // If Sequencing Transaction is in progress then currentWSID has non-zero value.
  // Cleared by s.Actualize()
  currentWSID   WSID
  currentWSKind WSKind

  // Set by s.actualizer()
  // Closed when flusher needs to be stopped
  flusherCtxCancel context.CancelFunc
  // Used to wait for flusher goroutine to exit
  // Set to nil when flusher is not running
  // Is not accessed concurrently since
  // - Is accessed by actualizer() and cleanup()
  // - cleanup() first shutdowns the actualizer() then flusher()
  flusherWG  *sync.WaitGroup
  // Buffered channel [1] to signal flusher to flush
  // Written (non-blocking) by Flush()
  flusherSig chan struct{}

  // To be flushed
  toBeFlushed map[NumberKey]Number
  // Will be 6 if the offset of the last processed event is 4
  toBeFlushedOffset PLogOffset
  // Protects toBeFlushed and toBeFlushedOffset
  toBeFlushedMu sync.RWMutex

  // Written by Next()
  inproc map[NumberKey]Number

}

// New creates a new instance of the Sequencer type.
// Instance has actualizer() goroutine started.
// cleanup: function to stop the actualizer.
func New(*isequencer.Params) (isequencer.ISequencer, cleanup func(), error) {
  // ...
}

// Flush implements isequencer.ISequencer.Flush.
// Flow:
//   Copy s.inproc and s.nextOffset to s.toBeFlushed and s.toBeFlushedOffset
//   Clear s.inproc
//   Increase s.nextOffset
//   Non-blocking write to s.flusherSig
func (s *sequencer) Flush() {
  // ...
}

// Next implements isequencer.ISequencer.Next.
// It ensures thread-safe access to sequence values and handles various caching layers.
//
// Flow:
// - Validate sequencing Transaction status
// - Get initialValue from s.params.SeqTypes and ensure that SeqID is known
// - Try to obtain the next value using:
//   - Try s.lru (can be evicted)
//   - Try s.inproc
//   - Try s.toBeFlushed (use s.toBeFlushedMu to synchronize)
//   - Try s.params.SeqStorage.ReadNumber()
//      - Read all known numbers for wsKind, wsID
//        - If number is 0 then initial value is used
//      - Write all numbers to s.lru
// - Write value+1 to s.lru
// - Write value+1 to s.inproc
// - Return value
func (s *sequencer) Next(seqID SeqID) (num Number, err error) {
  // ...
}

// batcher processes a batch of sequence values and writes maximum values to storage.
//
// Flow:
// - Wait until len(s.toBeFlushed) < s.params.MaxNumUnflushedValues
//   - Lock/Unlock
//   - Wait s.params.BatcherDelay
//   - check ctx (return ctx.Err())
// - s.nextOffset = offset + 1
// - Store maxValues in s.toBeFlushed: max Number for each SeqValue.Key
// - s.toBeFlushedOffset = offset + 1
//
func (s *sequencer) batcher(ctx ctx.Context, values []SeqValue, offset PLogOffset) (err error) {
  // ...
}

// Actualize implements isequencer.ISequencer.Actualize.
// Flow:
// - Validate Actualization status (s.actualizerInProgress is false)
// - Set s.actualizerInProgress to true
// - Set s.actualizerCtxCancel, s.actualizerWG
// - Start the actualizer() goroutine
func (s *sequencer) Actualize() {
  // ...
}

/*
actualizer is started in goroutine by Actualize().

Flow:

- if s.flusherWG is not nil
  - s.cancelFlusherCtx()
  - Wait for s.flusherWG
  - s.flusherWG = nil
- Clean s.lru, s.nextOffset, s.currentWSID, s.currentWSKind, s.toBeFlushed, s.inproc, s.toBeFlushedOffset
- s.flusherWG, s.flusherCtxCancel + start flusher() goroutine
- Read nextPLogOffset from s.params.SeqStorage.ReadNextPLogOffset()
- Use s.params.SeqStorage.ActualizeSequencesFromPLog() and s.batcher()

ctx handling:
 - if ctx is closed exit

Error handling:
- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) actualizer(ctx context.Context) {
  // ...
}

/*
flusher is started in goroutine by actualizer().

Flow:

- Wait for ctx.Done() or s.flusherSig
- if ctx.Done() exit
- Lock s.toBeFlushedMu
- Copy s.toBeFlushedOffset to flushOffset (local variable)
- Copy s.toBeFlushed to flushValues []SeqValue (local variable)
- Unlock s.toBeFlushedMu
- s.params.SeqStorage.WriteValues(flushValues, flushOffset)
- Lock s.toBeFlushedMu
- for each key in flushValues remove key from s.toBeFlushed if values are the same
- Unlock s.toBeFlushedMu

Error handling:

- Handle errors with retry mechanism (500ms wait)
- Retry mechanism must check `ctx` parameter, if exists
*/
func (s *sequencer) flusher(ctx context.Context) {
  // ...
}

// cleanup stops the actualizer() and flusher() goroutines.
// Flow:
// - if s.actualizerInProgress
//   - s.cancelActualizerCtx()
//   - Wait for s.actualizerWG
// - if s.flusherWG is not nil
//   - s.cancelFlusherCtx()
//   - Wait for s.flusherWG
//   - s.flusherWG = nil
func (s *sequencer) cleanup() {
  // ...
}
```

---

### ISeqStorage implementation

ISeqStorageImplementation: Implementation of the `isequencer.ISeqStorage` interface

- Package: appparts.internal.seqStorage
- ISeqStorageImplementation.New: Per App per Partition by AppParts
  - PartitionID is not passed to the constructor
- ReadValues: Handle [#688: record ID leads to different tables](https://github.com/voedger/voedger/issues/688)
  - If existing number is less than ??? 322_680_000_000_000 - do not send it to the batcher
- Uses VVMSeqStorage Adapter

### VVMStorage Adapter

VVMSeqStorageAdapter: Adapter that reads and writes sequence data to the VVMStorage

- PLogOffset in Partition storage: ((pKeyPrefix_SeqStorage_Part, PartitionID) PLogOffsetCC(0))
  - KeyPrefixSeqStoragePart
  - KeyPrefixSeqStorageWS
  - PLogOffsetCC
- Numbers: ((pKeyPrefix_SeqStorage_WS, AppID, WSID) SeqID)

---

## Technical design: Tests

### Integration tests for SequencesTrustLevel mode

Method:

- Test for Record
  - Create a new VIT instance on an owned config with `VVMConfig.TrustedSequences = false`
  - Insert a doc to get the last recordID: simply exec `c.sys.CUD` and get the ID of the new record
  - Corrupt the storage: Insert a conflicting key that will be used on creating the next record:
    - `VIT.IAppStorageProvider.AppStorage(test1/app1).Put()`
    - Build `pKey`, `cCols` for the record, use just inserted recordID+1
    - Value does not matter, let it be `[]byte{1}`
  - Try to insert one more record using `c.sys.CUD`
  - Expect panic
- Test for PLog, WLog offsets - the same tests but sabotage the storage building keys for the event

Tests:

- SequencesTrustLevel0: Integration test for `SequencesTrustLevel = 0`
- SequencesTrustLevel1: Integration test for `SequencesTrustLevel = 1`
- SequencesTrustLevel2: Integration test for `SequencesTrustLevel = 2`

### Integration tests for built-in sequences

- BuiltInSequences: Test for initial values: PLogOffsetSequence, WLogOffsetSequence, CRecordIDSequence, OWRecordIDSequence

---

## Addressed issues

- [#3215: Sequences](https://github.com/voedger/voedger/issues/3215) - Initial requirements and discussion

## References

Design process:

- [Voedger Sequence Management Design (Claude 3.7 Sonnet, March 1, 2025)](https://claude.ai/chat/f1a8492a-8e8a-4229-ac79-ecc3655732d3)

History:

- [Initial design](https://github.com/voedger/voedger-internals/blob/2475814f7caa1d2d400a62a788ceda9b16d8de2a/server/design/sequences.md)
