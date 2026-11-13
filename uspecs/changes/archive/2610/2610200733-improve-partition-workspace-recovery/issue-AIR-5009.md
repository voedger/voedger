# voedger: imrpove partition and workspace recovery

- URL: https://untill.atlassian.net/browse/AIR-5009
- ID: AIR-5009
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Parent: [AIR-4959: voedger: partition recovery performance](https://untill.atlassian.net/browse/AIR-4959)

## Partition recovery

- Actualize sequence counters in memory
  - Read SeqStorage_Part_PLog_offset
  - Actualize SeqStorage_Part_PLog_offset **in memory**
    - If SeqStorage_Part_PLog_offset == 0 then actualize vvm/storage[SeqStorage_WS_sequences] **in memory**
      - WLog.offset
      - records ids (nextRecordID), json (extensibility)
- Re-apply last event
- Save (order is important)
  - vvm/storage[SeqStorage_WS_sequences]
  - SeqStorage_Part_PLog_offset

## Sequence actualization

- Two async projectors, one per SeqStorage_Part_PLog_offset, SeqStorage_WS_sequences
- SeqStorage_Part_PLog_offset
  - Each 100th event  or 1 minute whichever comes first
- SeqStorage_WS_sequences: per ws: each operation
- Do NOT use events to compute counters, use counters from memory (from CP data structures)
  - Send data from CP in workpiece?

## Workspace recovery

- NumWSRecoverers: number of ws recovery goroutines per partition. Default 4.
- CP: When command comes for an unrecovered partition then partition recovery starts
- CP: When command comes for an recovered partition and unrecovered workspace - workspace recovery started (if not yet), client gets 503
  - Max NumWSRecoverers
- Update SeqStorage_WS_sequences

## Storages

- SeqStorage_Part_PLog_offset
  - pkey - appid, PartitionID
  - value: json: PLogOffset
- SeqStorage_WS_sequences
  - pkey: appid, WSID
  - value: json: WLogOffset, RecordIDs
