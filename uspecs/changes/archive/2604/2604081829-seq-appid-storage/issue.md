# AIR-3530: sequences: AppID is not considered in seqStorage

- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com

## Why

If Sequencer issues numbers for 2 different applications then it will be 1 and 2 whereas it should be 1 and 1 because app data is stored in different keyspaces. That happens because `IVVMSeqStorageAdapter` methods do not consider AppID.

## What

- Consider AppID in `IVVMSeqStorageAdapter`
- Make according updates to the sequences design
- Implement comprehensive tests to guard that fix
