Feature: Partition and workspace recovery

  VAOperator starts or restarts a VVM without losing the next partition-log
  offset, workspace-log offset, or workspace record ID, even when recovery
  checkpoints are absent, valid, or invalid, provided the logs retain the
  required allocation information. Marking the last ID-bearing WLog event as
  corrupted can remove that information even when its records remain stored.

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

  "The server has become ready" means bootstrap, including initial recovery,
  has completed. Scenarios need not repeat initial recovery as a precondition.
  This does not imply that request-driven partition or workspace recovery has
  completed, or that every declared partition was deployed.

  Glossary:
    Initial recovery: A VVM startup phase that creates missing partition and
      workspace recovery checkpoints before application partitions are deployed.
      A partition with an existing partition checkpoint, including zero, is skipped.
      It does not restore numbers for partition or workspace state; that recovery
      happens when commands arrive.

    ID-bearing event: A logged event that allocates at least one new ordinary
      (non-singleton) record ID. These IDs can belong to any docs including
      ODocs and nested docs.

    Non-ID-bearing event: A logged event that allocates no new ordinary record IDs.
      It can update existing records or insert singletons, provided its logged
      arguments also allocate no ordinary IDs.

    Invalid partition checkpoint: A stored nonzero PLog offset that does not
      identify an existing event, because that offset is missing or beyond the
      log end. An absent or zero checkpoint is not invalid.

    Invalid workspace checkpoint: A stored nonzero WLog offset that does not
      identify an existing ID-bearing event. The offset can
      be missing, beyond the log end, or point to a non-ID-bearing event.
      An absent or zero checkpoint is not invalid.

    Partition reset: Discarding the Command processor's in-memory recovery state
      for a partition due failure of any of the following operations:
      - PutPLog;
      - ApplyRecords;
      - sync projectors run;
      - PutWLog.
      The next command starts partition recovery again from persisted logs.
      Reset deletes no persisted data or checkpoints and does not roll back changes
      already written by the failed command. Reset requests cooperative cancellation
      of that partition instance's workspace recoveries without waiting for them.
      Canceled workers still count toward the workspace recovery limit until they exit.

    Workspace recovery limit: The maximum number of workspaces in one partition
      that can recover at the same time. Hardcoded to 4. A request to another
      unrecovered workspace is rejected without queueing when all slots are occupied
      and can be retried after a slot becomes available.

  Rule: Initial recovery

    The runtime-recovery scenarios under Partition recovery orchestration,
    Partition recovery, Workspace recovery, Empty WLog, and Workspace recovery
    orchestration apply to deployed partitions. Commands to undeployed partitions
    do not enter that lifecycle.

    Scenario: Normal initial partition recovery
      Given an application declares partition IDs 0 through 2
      And partition 0 has a stored partition checkpoint at its PLog end
      And the saved position of sys.ApplyRecoveryCheckpoint in partition 0 is FirstOffset
      And partitions 1 and 2 have no stored partition checkpoints
      And each partition is populated with a workspace having this history:
        | partition ID | workspace | last WLog offset | last allocated ordinary record ID |
        | 0            | Orders    | 7                | 200003                            |
        | 1            | Billing   | 14               | 200005                            |
        | 2            | Inventory | 20               | 200009                            |
      When VAOperator starts the VVM and initial recovery succeeds
      And after partition deployment a client inserts one ordinary record into each workspace, retrying "503 Service Unavailable" responses
      Then immediately after initial recovery, before the actualizer starts, the saved position of sys.ApplyRecoveryCheckpoint in partition 0 is still FirstOffset
      And each command eventually returns "200 OK"
      And the successful command responses contain:
        | workspace | CurrentWLogOffset | new ordinary record ID |
        | Orders    | 8                 | 200004                 |
        | Billing   | 15                | 200006                 |
        | Inventory | 21                | 200010                 |

    Scenario Outline: Initial recovery failures are logged while the server becomes ready
      Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
      And each partition has a nonempty PLog and no saved position for sys.ApplyRecoveryCheckpoint
      And the initial recovery operation "<operation>" fails for <failed_partitions> with "<cause>"
      When VAOperator starts the VVM
      Then each failed partition is undeployed and has no partition checkpoint
      And each remaining partition is deployed with a partition checkpoint
      And for each failed partition an ERROR log at stage "bootstrap.initial_recovery.error" identifies app "test1/app1" and its partition ID
      And the log message identifies "<operation>" and contains "<cause>"
      And no failed partition has an INFO log at stage "bootstrap.initial_recovery" with a message starting with "ap.sys.ApplyRecoveryCheckpoint offset set to"
      And the server becomes ready

      Examples:
        | operation                | failed_partitions      | cause               |
        | read PLog                | partition 1            | PLog read failed    |
        | read PLog                | partitions 0 through 2 | PLog read failed    |
        | read projector position  | partition 1            | storage unavailable |
        | write projector position | partition 1            | storage unavailable |

    Scenario: Commands return 500 on a partition whose initial recovery failed
      Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
      And workspace Billing belongs to partition 1
      And initial recovery fails for partition 1 with "PLog read failed"
      And VAOperator has started the VVM
      When a client inserts one ordinary record into Billing without retries
      Then the command returns "500 Internal Server Error"
      And its response error is "partition not found"

    Scenario Outline: The router receives the full partition count from the application descriptor despite initial recovery failures
      Given application "test1/app1" has a deployment descriptor with NumPartitions 3
      And workspace 10 has existing data in partition 1
      And initial recovery fails for <failed_partitions>
      And VAOperator has started the VVM
      When a client submits a command to workspace 10 through the router
      Then the client receives "500 Internal Server Error"
      And response error is "partition not found"

      Examples:
        | failed_partitions      |
        | partition 1            |
        | partitions 0 through 2 |

    Scenario: A changed app descriptor partition count causes panic
      Given application "test1/app1" has a persisted deployment descriptor with NumPartitions 3
      And all declared partitions are deployed
      When VAOperator restarts the VVM with a configured descriptor declaring NumPartitions 2
      Then the VVM launch panics because the descriptor partition count has changed
      And the persisted deployment descriptor still has NumPartitions 3
      And the application is not started with a different workspace placement

    Scenario: VVM launch fails with an error on failure of the initial recovery of a partition of the Cluster app
      Given initial recovery fails for the cluster application's sole partition
      When VAOperator starts the VVM
      Then VVM launch fails with an error

    Scenario Outline: A later bootstrap deploys a previously failed partition after successful initial recovery
      Given initial recovery of partition 1 of application "test1/app1" failed during "<operation>" on the previous VVM start
      And the partition checkpoint and the saved position for sys.ApplyRecoveryCheckpoint are absent
      And workspace Billing in partition 1 has last WLog offset 14 and last allocated ordinary record ID 200005
      And the storage failure has been resolved
      When VAOperator starts the VVM again
      And a client inserts one ordinary record into Billing after readiness, retrying "503 Service Unavailable" responses
      Then partition 1 is deployed with a partition checkpoint
      And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006

      Examples:
        | operation                |
        | read PLog                |
        | read projector position  |
        | write projector position |

  Rule: Recovery checkpoint projector

    Scenario Outline: Initial recovery sets the position of sys.ApplyRecoveryCheckpoint to the recovered PLog tail before the actualizer starts
      Given the PLog ends at offset 250 and the partition checkpoint is absent
      And the saved position of sys.ApplyRecoveryCheckpoint before initial recovery is <position>
      When initial recovery completes successfully for this partition
      Then immediately after initial recovery, before the actualizer starts, the saved position of sys.ApplyRecoveryCheckpoint is 250
      And the position lookup used by the actualizer returns 250 for this projector and partition
      And the partition checkpoint has lastPLogOffset 250
      And exactly one INFO log for this application and partition at stage "bootstrap.initial_recovery" has message "ap.sys.ApplyRecoveryCheckpoint offset set to 250"

      Examples:
        | position    |
        | absent      |
        | 0           |
        | FirstOffset |

    Scenario: The first insert after partition deployment receives the next record ID and WLog offset
      Given initial recovery completed at PLog offset 250 and the partition was deployed with sys.ApplyRecoveryCheckpoint at position 250
      And workspace "Orders" in that partition has last WLog offset 14 and last allocated ordinary record ID 200005
      When a client submits the partition's first command to insert one ordinary record into "Orders", retrying "503 Service Unavailable" responses
      Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
      And the insert appends PLog event 251
      And the workspace checkpoint for "Orders" eventually advances to wlogIDBearingOffset 15

    Scenario: The recovery checkpoint projector overwrites the initial recovery checkpoint without changing numbers issued by the next insert
      Given the PLog ends at offset 250 and the partition checkpoint is absent
      And the saved position of sys.ApplyRecoveryCheckpoint is 2
      And workspace "Orders" in that partition has last WLog offset 14 and last allocated ordinary record ID 200005
      When the following lifecycle occurs in order:
        | order | action                                                                                             |
        | 1     | VAOperator starts the VVM and initial recovery succeeds                                            |
        | 2     | sys.ApplyRecoveryCheckpoint catches up through PLog offset 250                                     |
        | 3     | VAOperator restarts the VVM                                                                        |
        | 4     | VAOperator inserts one ordinary record into "Orders", retrying "503 Service Unavailable" responses |
      Then immediately after initial recovery, before the actualizer starts, the saved position of sys.ApplyRecoveryCheckpoint is still 2 and the partition checkpoint has lastPLogOffset 250
      And this partition has no INFO log at stage "bootstrap.initial_recovery" with a message starting with "ap.sys.ApplyRecoveryCheckpoint offset set to"
      And the projector's partition checkpoint writes during catch-up are:
        | processed PLog offset | stored lastPLogOffset |
        | 100                   | 100                   |
        | 200                   | 200                   |
      And the stored partition checkpoint is lastPLogOffset 200
      And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
      And the insert writes the next PLog event at offset 251

    Scenario: Initial recovery marks an empty partition complete without creating a projector position
      Given the PLog is empty and the partition checkpoint is absent
      And sys.ApplyRecoveryCheckpoint has no saved position
      When VAOperator starts the VVM
      Then the stored partition checkpoint is lastPLogOffset 0
      And the saved position of sys.ApplyRecoveryCheckpoint is absent

  Rule: Partition recovery orchestration

    Scenario: At least the first two command requests to a partition and workspace return 503 after initial recovery
      Given the server has become ready
      When VAOperator submits command requests to a workspace in partition 1 in this order:
        | request | timing               | retry policy            |
        | 1       | after readiness      | none                    |
        | 2       | after response 1     | none                    |
        | 3       | after response 2     | 503 Service Unavailable |
      Then the responses are:
        | request | status                  |
        | 1       | 503 Service Unavailable |
        | 2       | 503 Service Unavailable |
        | 3       | 200 OK                  |

    Scenario: A query does not trigger partition recovery
      Given the server has become ready
      When VAOperator submits the very first query request to an existing workspace in any deployed partition without automatic retries
      Then response status is "200 OK"
      And no partition or workspace recovery is started by the query

    Scenario: Concurrent requests to the same partition start only one recovery
      # Request 1 starts recovery of partition 1; requests 2 through 6 arrive before it finishes
      # All six receive 503, but only one recovery runs; request 7 succeeds after recovery
      Given partition 1 is deployed but unrecovered
      And workspace 1 has only its initialization event at WLog offset 1 and no user record IDs
      When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
        | request     | timing                                   | retry policy            |
        | 1           | before partition recovery starts         | none                    |
        | 2 through 6 | concurrently while recovery is running   | none                    |
        | 7           | after partition recovery completes       | 503 Service Unavailable |
      Then the responses are:
        | request     | status                  |
        | 1 through 6 | 503 Service Unavailable |
        | 7           | 200 OK                  |
      And response 7 contains CurrentWLogOffset 2 and new record IDs 200001 through 200003
      And error stage "cp.error" logs message "partition 1: recovery is in progress"
      And exactly one partition-recovery start is logged for partition 1

    Scenario: A command succeeds in another partition while partition recovery is running
      Given partition 1 recovery is in progress
      And workspace 2 in partition 0 has only its initialization event at WLog offset 1 and no user record IDs
      When VAOperator inserts an ordinary record into workspace 2, retrying "503 Service Unavailable" responses
      Then the insert returns "200 OK" with CurrentWLogOffset 2 and new ordinary record ID FirstUserRecordID
      And partition 1 recovery has not completed when the insert succeeds

    Scenario: Canceling the initiating request does not stop partition recovery
      Given partition 1 recovery was started by a command request to workspace 1
      And workspace 1 has only its initialization event at WLog offset 1 and no user record IDs
      When VAOperator cancels the initiating request
      Then inserting three ordinary records into workspace 1 after recovery completes, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 2 and new record IDs 200001 through 200003
      And exactly one partition-recovery start is logged for partition 1

    Scenario: An unauthenticated command does not trigger partition recovery
      Given partition 1 is deployed but unrecovered
      When VAOperator submits a command to workspace 1 with an invalid token without automatic retries
      Then response status is "401 Unauthorized"
      And no partition or workspace recovery is started by the command

    Scenario: The next request restarts recovery of the failed partition and report the error
      Given partition 1 is deployed but unrecovered
      And workspace 1 has only its initialization event at WLog offset 1 and no user record IDs
      And its first partition recovery fails with "PLog read failed"
      When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
        | request | timing                               | retry policy            |
        | 1       | before partition recovery starts     | none                    |
        | 2       | after the first recovery fails       | none                    |
        | 3       | while the new attempt is in progress | none                    |
        | 4       | after the new attempt completes      | 503 Service Unavailable |
      Then the responses are:
        | request | status                    |
        | 1       | 503 Service Unavailable   |
        | 2       | 500 Internal Server Error |
        | 3       | 503 Service Unavailable   |
        | 4       | 200 OK                    |
      And request 2 returns error "partition 1: recovery failed: PLog read failed" and starts one new recovery attempt
      And request 3 does not start another recovery attempt
      And the fourth response contains CurrentWLogOffset 2 and new record IDs 200001 through 200003

  Rule: Partition recovery

    usable checkpoint ──────────────────> resume from checkpoint ──┐
    absent or zero checkpoint ──────────> recover from FirstOffset ├─> actual PLog end ──> later command succeeds after workspace recovery
    invalid checkpoint ─> log error ────> recover from FirstOffset ┘

    An absent or zero partition checkpoint starts one scan at FirstOffset.

    Scenario Outline: Normal partion recovery from checkpoint
      Given the PLog contains events through offset 50
      And workspace 1 has last WLog offset 14 and last allocated ordinary record ID 200005
      And the stored partition checkpoint is <checkpoint>
      And the Command processor has restarted
      When VAOperator submits an insert into the unrecovered partition, retrying "503 Service Unavailable" responses
      Then Partition recovery calls ReadPLog once from offset <startOffset> through the log end
      And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006

      Examples:
        | checkpoint        | startOffset |
        | lastPLogOffset 10 | 10          |
        | lastPLogOffset 50 | 50          |
        | lastPLogOffset 0  | 1           |
        | absent            | 1           |


    Scenario Outline: Invalid partition checkpoint causes PLog read out from beginning
      Given the PLog event offsets are <plog>
      And workspace "recovery-test-workspace" has last WLog offset <lastWLogOffset> and record ID allocation baseline <lastRecordID>
      And the stored partition checkpoint is <lastPLogOffset>
      When VAOperator restarts the VVM
      Then inserting an ordinary record into "recovery-test-workspace", retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID <lastRecordID> + 1

      Examples:
        | plog            | lastPLogOffset | lastWLogOffset | lastRecordID |
        | 1..50           | 1000000        | 14             | 200005       |
        | 2..50           | 1              | 14             | 200005       |
        | 1..50 except 10 | 10             | 14             | 200005       |

    Scenario Outline: Empty PLog recovery preserves initial sequence numbers
      Given the PLog is empty
      And the stored partition checkpoint is <checkpoint>
      When VAOperator restarts the VVM
      Then inserting an ordinary record into a workspace, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 1 and new ordinary record ID 200001

      Examples:
        | checkpoint       |
        | absent           |
        | lastPLogOffset 0 |
        | lastPLogOffset 5 |

    Scenario: Replaying records from a corrupted final PLog event after restart panics
      Given the final PLog event of a deployed partition belongs to workspace "Orders"
      And VAOperator has marked that event as corrupted
      And the VVM has restarted and the server has become ready
      When the final PLog event's records are replayed
      Then record replay panics with "event is not valid: can not apply not valid event: corrupted data:"

  Rule: Workspace recovery

    An absent or zero workspace checkpoint starts one scan at FirstOffset.

    Scenario Outline: Normal workspace recovery from checkpoint
      Given the WLog has an ID-bearing event at offset 6 with record ID 200001
      And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
      And any WLog events after offset 10 are CUD-free
      And the WLog ends at offset <lastWLogOffset>
      And the stored workspace checkpoint is <checkpoint>
      And the Command processor has restarted and completed Partition recovery
      When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
      Then Workspace recovery calls ReadWLog once from offset <startOffset> through the log end
      And the insert returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID 200006

      Examples:
        | checkpoint             | lastWLogOffset | startOffset |
        | wlogIDBearingOffset 6  | 14             | 6           |
        | wlogIDBearingOffset 10 | 14             | 10          |
        | wlogIDBearingOffset 10 | 10             | 10          |
        | wlogIDBearingOffset 0  | 14             | 1           |
        | absent                 | 14             | 1           |

    Scenario: Workspace recovery preserves sequence numbers after a mixed non-ID-bearing WLog tail
      Given the WLog has an ID-bearing event at offset 6 with record ID 200001
      And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
      And the stored workspace checkpoint is wlogIDBearingOffset 10
      And the WLog has non-CUD events at offsets 11 and 12
      And the WLog has a singleton insert at offset 13
      And the WLog has an update of record 200002 at offset 14
      And the WLog ends at offset 14
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an ordinary record insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
      Then the insert returns "200 OK" with CurrentWLogOffset 15 and NewIDs {"1": 200006}

    Scenario: Workspace without user record IDs starts allocation at FirstUserRecordID
      Given the WLog has three non-ID-bearing events at offsets 1 through 3
      And the stored workspace checkpoint is wlogIDBearingOffset FirstOffset
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an ordinary record insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
      Then the insert returns "200 OK" with CurrentWLogOffset 4 and new ordinary record ID FirstUserRecordID (200001)

    Scenario Outline: Invalid workspace checkpoint falls back to the WLog beginning
      # The checkpoint either misses an existing event or points to one that allocates no ordinary IDs
      # Recovery scans from FirstOffset to find the actual WLog end and highest allocated ID
      # The next insert therefore uses WLog offset 15 and record ID 200006, regardless of the invalid checkpoint
      Given the WLog has an ID-bearing event at offset 6 with record ID 200001
      And the WLog has a non-ID-bearing event at offset 9
      And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
      And the WLog has a CUD-free event at offset 12
      And the WLog has a singleton-only event at offset 13
      And the WLog has an update of an existing record at offset 14
      And the WLog ends at offset 14
      And the stored workspace checkpoint is <checkpoint>
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
      Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
      And an ERROR log at stage "cp.workspace_recovery.checkpoint.invalid" identifies the stored WLog offset
      And its message contains "does not identify an ID-bearing event; retrying from FirstOffset"

      Examples:
        | checkpoint                                                   |
        | wlogIDBearingOffset 15 beyond the WLog end                   |
        | wlogIDBearingOffset 8 whose first returned event is 9        |
        | wlogIDBearingOffset 12 pointing to a CUD-free event          |
        | wlogIDBearingOffset 13 pointing to a singleton-only event    |
        | wlogIDBearingOffset 14 pointing to an existing-record update |

    Scenario: An insert fails after recovery when the last ID-bearing WLog event is corrupted
      Given sequence trust level is 0 (do not trust new record writes)
      And workspace "Orders" has this complete WLog history before corruption:
        | WLog offset | command                  | allocated ordinary record ID |
        | 1 through 3 | Workspace initialization | none                         |
        | 4           | Ordinary record insert   | 200001                       |
        | 5           | Ordinary record insert   | 200002                       |
        | 6           | Non-ID-bearing command   | none                         |
      And its partition's last PLog event before restart is the event at WLog offset 6 in "Orders"
      And VAOperator has marked WLog event 5 in "Orders" as corrupted
      And ordinary record 200002 remains stored in "Orders"
      And no events after WLog offset 5 in "Orders" allocate ordinary record IDs
      And the VVM has restarted and completed partition and workspace recovery for "Orders"
      When VAOperator inserts one ordinary record into "Orders" without automatic retries
      Then the insert returns "500 Internal Server Error"
      And the response error contains "sequences violation"

  Rule: Empty WLog

    Invalid workspace checkpoints are reported even when the WLog is empty.

    Scenario: Invalid workspace checkpoint over an empty WLog
      Given the WLog is empty
      And the stored workspace checkpoint is wlogIDBearingOffset 5
      And the VVM has restarted and completed partition recovery
      When VAOperator submits an ordinary record insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
      Then the insert returns "200 OK" with CurrentWLogOffset FirstOffset and new ordinary record ID FirstUserRecordID
      And error stage "cp.workspace_recovery.checkpoint.invalid" is logged
      And error message is "stored WLog offset 5 does not identify an ID-bearing event; retrying from FirstOffset"

  Rule: Workspace recovery orchestration

    request ── ws is recovering now ──────> shares the running recovery
    ├─ recovery limit available ──────────> starts one recovery
    └─ recovery limit reached ────────────> rejected without queueing
    retained failure ── next request ─────> reports the failure and starts one retry

    Scenario: Concurrent requests to the same workspace start only one recovery
      # Request 1 starts recovery of workspace 42; request 2 arrives before it finishes
      # Both receive 503 without automatic retries, and request 2 starts no additional recovery
      # After recovery completes, request 3 inserts record 200006 at WLog offset 15
      Given workspace 42 is unrecovered after partition recovery
      And its WLog ends at offset 14 with last user record ID 200005
      When VAOperator submits ordinary record inserts into workspace 42 without automatic retries at these times:
        | request | timing                                   |
        | 1       | before workspace recovery starts         |
        | 2       | while that recovery is still in progress |
        | 3       | after that recovery completes            |
      Then the responses are:
        | request | status                  |
        | 1       | 503 Service Unavailable |
        | 2       | 503 Service Unavailable |
        | 3       | 200 OK                  |
      And the third response contains CurrentWLogOffset 15 and new ordinary record ID 200006
      And error stage "cp.error" logs message "workspace 42: recovery is in progress"
      And exactly one workspace-recovery start is logged for workspace 42

    Scenario: Num of workspaces recovering at the same time is limited per partition
      Given workspaces 42 and 43 are unrecovered after partition recovery
      And both WLogs end at offset 14 with last user record ID 200005
      And workspace recoveries for workspaces 44 through 46 are running
      When VAOperator submits inserts in this order:
        | request | workspace | timing                                 |
        | 1       | 42        | while workspaces 44 through 46 recover |
        | 2       | 43        | before workspace 42 recovery completes |
        | 3       | 43        | after workspace 42 recovery completes  |
        | 4       | 43        | after workspace 43 recovery completes  |
      Then the responses are:
        | request | status                  |
        | 1       | 503 Service Unavailable |
        | 2       | 503 Service Unavailable |
        | 3       | 503 Service Unavailable |
        | 4       | 200 OK                  |
      And request 2 logs message "workspace 43: recovery concurrency limit is reached" at error stage "cp.error"
      And no workspace-recovery start is logged for workspace 43 before request 3
      And the fourth response contains CurrentWLogOffset 15 and new ordinary record ID 200006

    Scenario: A partition reset cancels current workspace recoveries without waiting for them to finish
      Given workspace 1 is recovered
      And workspaces 3 and 5 are recovering from WLogs in partition 1 of "untill/airs-bp"
      And their reads observe cancellation but wait for replacement partition recovery
      And a synchronous projector will fail the next workspace 1 command with "projector failed"
      When VAOperator inserts into workspace 1 without retries:
        | request | timing                                |
        | 1       | during WLog reads                     |
        | 2       | after response 1, before reads return |
      Then the responses are:
        | request | status                    | error                                |
        | 1       | 500 Internal Server Error | projector failed                     |
        | 2       | 503 Service Unavailable   | partition 1: recovery is in progress |
      And replacement partition recovery completes before the reads return
      And each canceled recovery logs INFO once at "cp.workspace_recovery.canceled"
      And each log has a message starting with "partition reset, duration ", app "untill/airs-bp", partid 1, and its wsid
      And canceled recoveries log neither "cp.workspace_recovery.complete" nor "cp.workspace_recovery.readwlog.error"

    Scenario: The next request restarts recovery of the failed workspace and report the error
      Given workspace 42 is unrecovered after partition recovery
      And its WLog ends at offset 14 with last user record ID 200005
      And its first workspace recovery fails with "WLog read failed"
      When VAOperator submits inserts into workspace 42 at these times:
        | request | timing                               |
        | 1       | before workspace recovery starts     |
        | 2       | after the first recovery fails       |
        | 3       | while the new attempt is in progress |
        | 4       | after the new attempt completes      |
      Then the responses are:
        | request | status                    |
        | 1       | 503 Service Unavailable   |
        | 2       | 500 Internal Server Error |
        | 3       | 503 Service Unavailable   |
        | 4       | 200 OK                    |
      And request 2 returns error "workspace 42: recovery failed: WLog read failed" and starts one new recovery attempt
      And request 3 does not start another recovery attempt
      And the fourth response contains CurrentWLogOffset 15 and new ordinary record ID 200006
