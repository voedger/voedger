Feature: Partition and workspace recovery

  VAOperator starts or restarts a VVM without losing the next partition-log
  offset, workspace-log offset, or workspace record ID, even when recovery
  checkpoints are absent, valid, or invalid.

  Bootstrap isolates an initial-checkpoint failure to its partition: other
  partition attempts finish and bootstrap continues. The server becomes ready
  even if every partition attempt for a non-cluster application fails. The
  cluster application's sole partition must initialize and deploy successfully,
  otherwise bootstrap returns an error and the server does not become ready.
  Failed partitions are not deployed and
  run no asynchronous projectors or scheduled jobs. Their commands return the
  standard partition-not-found error; initial-recovery failure details appear
  only in logs. A later bootstrap retries partitions without completion markers.

  Checkpoint failures do not change the descriptor's total partition count or
  workspace placement. Requests to unavailable partitions fail without being
  redirected to another partition.

  Every WLog event originates in the partition's PLog. If a partition's PLog is
  empty, all of its WLogs are empty too; recovery scenarios exclude an empty
  PLog paired with a non-empty WLog.

  Rule: Initial recovery

    The runtime-recovery scenarios under Partition checkpoint, Workspace
    checkpoint, Empty WLog, and Workspace recovery admission apply to deployed
    partitions. Commands to undeployed partitions do not enter that lifecycle.

    Scenario: Missing checkpoints are computed for every declared partition
      Given an application declares partition IDs 0 through 2
      And partition 0 has stored checkpoint lastPLogOffset 20
      And partition 1 has no stored partition checkpoint
      And partition 1 PLog ends at offset 50 and contains workspace histories:
        | WSID | latest ID-bearing WLog offset |
        | 10   | 6                             |
        | 11   | none                          |
      And partition 2 has no stored partition checkpoint
      And partition 2 PLog is empty
      When VAOperator starts the VVM
      Then partition 0 retains checkpoint lastPLogOffset 20
      And workspace checkpoints for partition 1 are persisted before its partition checkpoint:
        | WSID | wlogIDBearingOffset |
        | 10   | 6                   |
        | 11   | 0                   |
      And partition checkpoints are persisted:
        | partition ID | lastPLogOffset |
        | 1            | 50             |
        | 2            | 0              |
      And partitions 0 through 2 are deployed
      And the server becomes ready

    Scenario: Initial recovery initializes checkpoints but not runtime sequence state
      Given initial recovery has created checkpoints for partition 1 and workspace 10
      And the server has become ready
      When VAOperator submits a command to workspace 10
      Then response status is "503 Service Unavailable" while partition recovery runs
      When VAOperator retries the command after partition recovery completes
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the command after workspace recovery completes
      Then the command succeeds

    Scenario Outline: Initial recovery failures do not prevent server readiness
      Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
      And asynchronous projectors and scheduled jobs are configured for every partition
      And initial recovery fails for <failed_partitions> with "PLog read failed"
      When VAOperator starts the VVM
      Then partition checkpoints are <checkpoint_state>
      And initial-recovery error "PLog read failed" is logged for <failed_partitions>
      And deployed partitions are <deployed_partitions>
      And deployment is skipped for <failed_partitions>
      And no asynchronous projector or scheduled job runs for <failed_partitions>
      And the server becomes ready
      When VAOperator submits a command to partition <target_partition>
      Then response status is "500 Internal Server Error"
      And response error is "partition not found"
      When the PLog read failure is resolved without restarting the VVM
      And VAOperator submits another command to partition <target_partition>
      Then response status is "500 Internal Server Error"
      And response error is "partition not found"
      And no partition or workspace recovery has started for partition <target_partition>
      And no command event has been appended for partition <target_partition>
      And deployment is still skipped for <failed_partitions>
      And no asynchronous projector or scheduled job runs for <failed_partitions>

      Examples:
        | failed_partitions      | checkpoint_state                    | deployed_partitions | target_partition |
        | partition 1            | present for partitions 0 and 2 only | 0 and 2             | 1                |
        | partitions 0 through 2 | absent for every partition          | none                | 0                |
        | partitions 0 through 2 | absent for every partition          | none                | 1                |
        | partitions 0 through 2 | absent for every partition          | none                | 2                |

    Scenario Outline: Checkpoint failures preserve the descriptor partition count
      Given application "test1/app1" has a deployment descriptor with NumPartitions 3
      And workspace 10 has existing data in partition 1
      And initial recovery fails for <failed_partitions>
      When VAOperator starts the VVM
      Then the server becomes ready with deployed partitions <deployed_partitions>
      And the application's deployment descriptor still has NumPartitions 3
      And the application's advertised partition count is 3
      And workspace 10 remains assigned to partition 1
      When VAOperator submits a command to workspace 10 through the router
      Then response status is "500 Internal Server Error"
      And response error is "partition not found"
      And the command is not redirected to partition 0 or 2
      And workspace 10 data remains unchanged

      Examples:
        | failed_partitions      | deployed_partitions |
        | partition 1            | 0 and 2             |
        | partitions 0 through 2 | none                |

    Scenario: A changed descriptor partition count is rejected after partial deployment
      Given application "test1/app1" has a persisted deployment descriptor with NumPartitions 3
      And partition 1 was not deployed because its checkpoint initialization failed
      When VAOperator restarts the VVM with a configured descriptor declaring NumPartitions 2
      Then bootstrap fails with "409 Conflict" for the changed partition count
      And the persisted deployment descriptor still has NumPartitions 3
      And the application is not started with a different workspace placement

    Scenario: Cluster partition failure stops bootstrap
      Given initial recovery fails for the cluster application's sole partition
      When VAOperator starts the VVM
      Then bootstrap returns a cluster partition initialization or deployment error
      And the server does not become ready
      And no other application is registered or provisioned

    Scenario: Healthy partitions continue while a failed partition remains undeployed
      Given application "test1/app1" declares partition IDs 0 through 2
      And asynchronous projectors and scheduled jobs are configured for every partition
      And the VVM has started with initial recovery failed for partition 1 with "PLog read failed"
      And partitions 0 and 2 are deployed and the server is ready
      When VAOperator submits commands to partitions 0 and 2 until runtime recovery completes
      Then the commands succeed on partitions 0 and 2
      And partition 1 remains undeployed
      And no asynchronous projector or scheduled job runs for partition 1
      When VAOperator submits a command to partition 1
      Then response status is "500 Internal Server Error"
      And response error is "partition not found"

    Scenario: A later bootstrap deploys a previously failed partition after successful initial recovery
      Given application "test1/app1" declares partition IDs 0 through 2
      And the VVM has started with initial recovery failed for partition 1 with "PLog read failed"
      And partition 1 is undeployed and has no partition checkpoint
      And partitions 0 and 2 have stored partition checkpoints
      When the PLog read failure is resolved and VAOperator restarts the VVM
      Then partition 1 receives a partition checkpoint after its workspace checkpoints are persisted
      And the stored partition checkpoints for partitions 0 and 2 are unchanged
      And partitions 0 through 2 are deployed
      And the server becomes ready
      When VAOperator submits a command to partition 1 until runtime recovery completes
      Then the command succeeds

  Rule: Partition checkpoint

    usable checkpoint ──────────────────> resume from checkpoint ──┐
    absent or zero checkpoint ──────────> recover from FirstOffset ├─> actual PLog end ──> later command succeeds after workspace recovery
    invalid checkpoint ─> log error ────> recover from FirstOffset ┘

    An absent or zero partition checkpoint starts one scan at FirstOffset.
    Zero is an initialization sentinel, not an event offset to validate, so this
    path does not fall back or report an invalid checkpoint.

    Scenario Outline: Partition recovery starts from a checkpoint
      Given the PLog contains events through offset 50
      And the stored partition checkpoint is <checkpoint>
      When VAOperator restarts the VVM and submits a command
      Then response status is "503 Service Unavailable" while partition recovery runs
      When VAOperator retries the command after partition recovery completes
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the command after workspace recovery completes
      Then the command succeeds with a PLog event after offset 50
      And no invalid partition-checkpoint error is logged

      Examples:
        | checkpoint          |
        | absent              |
        | lastPLogOffset 0    |
        | lastPLogOffset 10   |
        | lastPLogOffset 50   |

    Scenario Outline: Invalid partition checkpoint falls back to the PLog beginning
      Given the PLog contains events through offset 50
      And the stored partition checkpoint is <checkpoint>
      When VAOperator restarts the VVM and submits a command
      Then response status is "503 Service Unavailable" while partition recovery runs
      When VAOperator retries the command after partition recovery completes
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the command after workspace recovery completes
      Then the command succeeds with a PLog event after offset 50
      And error stage "cp.partition_recovery.checkpoint.invalid" is logged
      And error message is "stored PLog offset <checkpoint_offset> does not identify an existing event; retrying from FirstOffset"

      Examples:
        | checkpoint                                         | checkpoint_offset |
        | lastPLogOffset 1000000 beyond the PLog end         | 1000000           |
        | lastPLogOffset 10 whose first returned event is 11 | 10                |

    Scenario Outline: Empty PLog recovery starts at FirstOffset
      Given the PLog is empty
      And the stored partition checkpoint is <checkpoint>
      When VAOperator restarts the VVM and submits a command
      Then response status is "503 Service Unavailable" while partition recovery runs
      When VAOperator retries the command after partition recovery completes
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the command after workspace recovery completes
      Then the command succeeds with a PLog event at FirstOffset
      And no invalid partition-checkpoint error is logged

      Examples:
        | checkpoint       |
        | absent           |
        | lastPLogOffset 0 |

    Scenario: Invalid partition checkpoint over an empty PLog
      Given the PLog is empty
      And the stored partition checkpoint is lastPLogOffset 5
      When VAOperator restarts the VVM and submits a command
      Then response status is "503 Service Unavailable" while partition recovery runs
      When VAOperator retries the command after partition recovery completes
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the command after workspace recovery completes
      Then the command succeeds with a PLog event at FirstOffset
      And error stage "cp.partition_recovery.checkpoint.invalid" is logged
      And error message is "stored PLog offset 5 does not identify an existing event; retrying from FirstOffset"

  Rule: Workspace checkpoint

    workspace checkpoint ──> recover actual WLog end ──────────────> successful retry uses following WLog offset
                          └─> recover latest allocated user record ID ──> successful retry uses following record ID
    non-ID-bearing log tail events ─────────────────────────────────> advances the WLog offset only

    An absent or zero workspace checkpoint starts one scan at FirstOffset.
    Zero is an initialization sentinel, not an event offset to validate, so this
    path does not fall back or report an invalid checkpoint.

    Scenario Outline: Workspace recovery starts from a checkpoint
      Given the WLog has an ID-bearing event at offset 6 with record ID 200001
      And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
      And the WLog has a CUD-free event at offset 14
      And the WLog ends at offset 14
      And the stored workspace checkpoint is <checkpoint>
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200006
      And no invalid workspace-checkpoint error is logged

      Examples:
        | checkpoint                |
        | absent                    |
        | wlogIDBearingOffset 0     |
        | wlogIDBearingOffset 6     |
        | wlogIDBearingOffset 10    |

    Scenario: Workspace recovery starts from an ID-bearing checkpoint at the WLog end
      Given the WLog has allocated record IDs 200001 through 200005
      And its ID-bearing event at offset 10 is the WLog end
      And the stored workspace checkpoint is wlogIDBearingOffset 10
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 11
      And the inserted record has ID 200006
      And no invalid workspace-checkpoint error is logged

    Scenario Outline: Non-ID-bearing log tail events advance only the WLog offset
      Given the WLog has allocated record IDs 200001 through 200005 at offset 10
      And the WLog has <tail_events> from offset 11 through offset 14
      And the stored workspace checkpoint is wlogIDBearingOffset 10
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200006
      And no invalid workspace-checkpoint error is logged

      Examples:
        | tail_events                 |
        | CUD-free events             |
        | singleton-only events       |
        | updates of existing records |

    Scenario Outline: Workspace without user record IDs starts allocation at FirstUserRecordID
      Given the WLog contains CUD-free and singleton-only events through offset 14
      And the WLog contains no newly allocated non-singleton record ID
      And the stored workspace checkpoint is <checkpoint>
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200001
      And no invalid workspace-checkpoint error is logged

      Examples:
        | checkpoint            |
        | absent                |
        | wlogIDBearingOffset 0 |

    Scenario Outline: Invalid workspace checkpoint falls back to the WLog beginning
      Given the WLog has an ID-bearing event at offset 6 with record ID 200001
      And the WLog has a non-ID-bearing event at offset 9
      And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
      And the WLog has a CUD-free event at offset 12
      And the WLog has a singleton-only event at offset 13
      And the WLog has an update of an existing record at offset 14
      And the WLog ends at offset 14
      And the stored workspace checkpoint is <checkpoint>
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200006
      And error stage "cp.workspace_recovery.checkpoint.invalid" is logged
      And error message is "stored WLog offset <checkpoint_offset> does not identify an ID-bearing event; retrying from FirstOffset"

      Examples:
        | checkpoint                                                   | checkpoint_offset |
        | wlogIDBearingOffset 15 beyond the WLog end                   | 15                |
        | wlogIDBearingOffset 8 whose first returned event is 9        | 8                 |
        | wlogIDBearingOffset 12 pointing to a CUD-free event          | 12                |
        | wlogIDBearingOffset 13 pointing to a singleton-only event    | 13                |
        | wlogIDBearingOffset 14 pointing to an existing-record update | 14                |

  Rule: Empty WLog

    absent or zero checkpoint ─────────────> first successful insert uses FirstOffset and FirstUserRecordID
    invalid checkpoint ─> specified error ──┘

    Scenario Outline: Empty WLog recovery starts at FirstOffset and FirstUserRecordID
      Given the WLog is empty
      And the stored workspace checkpoint is <checkpoint>
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at FirstOffset
      And the inserted record has ID 200001
      And no invalid workspace-checkpoint error is logged

      Examples:
        | checkpoint            |
        | absent                |
        | wlogIDBearingOffset 0 |

    Scenario: Invalid workspace checkpoint over an empty WLog
      Given the WLog is empty
      And the stored workspace checkpoint is wlogIDBearingOffset 5
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at FirstOffset
      And the inserted record has ID 200001
      And error stage "cp.workspace_recovery.checkpoint.invalid" is logged
      And error message is "stored WLog offset 5 does not identify an ID-bearing event; retrying from FirstOffset"

  Rule: Workspace recovery admission

    request ── duplicate ───────────────────> shares the running recovery
            ├─ recovery limit available ────> starts one recovery
            └─ recovery limit reached ──────> rejected without queueing
    retained failure ── next request ───────> reports the failure and starts one retry

    Scenario: Duplicate requests share one workspace recovery
      Given workspace 42 is unrecovered after partition recovery
      And its WLog ends at offset 14 with last user record ID 200005
      When VAOperator submits an insert into workspace 42
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator submits another insert into workspace 42 before recovery completes
      Then response status is "503 Service Unavailable" while the same recovery runs
      And error stage "cp.error" logs message "workspace 42: recovery is in progress"
      And exactly one workspace-recovery start is logged for workspace 42
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200006

    Scenario: Recovery limit rejects without queueing and permits a later retry
      Given the workspace recovery limit is 4
      And workspaces 42 and 43 are unrecovered after partition recovery
      And both WLogs end at offset 14 with last user record ID 200005
      And workspace recoveries for workspaces 44 through 46 are running
      When VAOperator submits an insert into workspace 42
      Then response status is "503 Service Unavailable" while workspace 42 recovery runs
      When VAOperator submits an insert into workspace 43 before workspace 42 recovery completes
      Then response status is "503 Service Unavailable" because the recovery limit is reached
      And error stage "cp.error" logs message "workspace 43: recovery concurrency limit is reached"
      And no workspace-recovery start is logged for workspace 43
      When workspace 42 recovery completes and VAOperator submits another insert into workspace 43
      Then response status is "503 Service Unavailable" while workspace 43 recovery runs
      When VAOperator retries the insert after workspace 43 recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200006

    Scenario: Partition reset preserves workspace recovery capacity
      Given the workspace recovery limit is 4
      And workspace 1 is recovered
      And workspace recoveries for workspaces 3, 5, 7, and 9 are running in the same partition
      When a command failure resets and recovers the partition
      And VAOperator submits an insert into unrecovered workspace 11
      Then response status is "503 Service Unavailable" because the recovery limit is reached
      And no workspace recovery starts for workspace 11
      When one detached workspace recovery completes
      And VAOperator submits another insert into workspace 11
      Then response status is "503 Service Unavailable" while workspace 11 recovery runs

    Scenario: Failed workspace recovery is reported and retried once
      Given workspace 42 is unrecovered after partition recovery
      And its WLog ends at offset 14 with last user record ID 200005
      And its first workspace recovery fails with "WLog read failed"
      When VAOperator submits an insert into workspace 42
      Then response status is "503 Service Unavailable" while workspace recovery runs
      When VAOperator submits another insert after that recovery fails
      Then response status is "500 Internal Server Error"
      And response error is "workspace 42: recovery failed: WLog read failed"
      And one retry of workspace 42 recovery starts
      When VAOperator submits another insert before that retry completes
      Then response status is "503 Service Unavailable" while the retry runs
      When VAOperator retries the insert after workspace recovery completes
      Then the insert succeeds with a WLog event at offset 15
      And the inserted record has ID 200006
