# voedger: resume sys.ApplyRecoveryCheckpoint from the PLog offset reached by initial recovery

- URL: https://untill.atlassian.net/browse/AIR-5369
- ID: AIR-5369
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

Initial recovery scans the entire PLog and stores the last offset as the partition checkpoint. On its first run, `sys.ApplyRecoveryCheckpoint` can replay the same log from `FirstOffset`, overwriting that checkpoint with older offsets. For example, if recovery stored `250`, the projector writes `100` and then `200` while catching up.

## What

After initial recovery persists the workspace checkpoints for a nonempty partition, and before it marks the partition complete, check the saved offset for `sys.ApplyRecoveryCheckpoint`. **Only if the saved offset is** `0` **(including a missing offset) or** `FirstOffset`, set it to the last PLog offset scanned. The actualizer then resumes at the next event. Preserve any projector offset greater than `FirstOffset`
