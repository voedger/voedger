# document that ApplyRecoveryCheckpoint projector will overwrite the initial checkpoint on first run

- URL: https://untill.atlassian.net/browse/AIR-5299
- ID: AIR-5299
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

Case

* Initial recovery is done
* Checkpoint pointed to offset 1000 is stored
* ApplyPartitionCheckpoint porjector is new and thus started to run over the log from the FirstOffset
* each 100 event the stored checkpoint (initially 1000) overwites with 100, then 200 etc

This case is not a bug, it is a normal behaviour. Overwritting the offset does not break the checkpoint bookkeepping. The last stored offset might be just descreased by a value between 0 and 100. This tail will be just read out on the recovery

## What

* cover by gherkin scenarios that behaviour
* implement corresponding tests

