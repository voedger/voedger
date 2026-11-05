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

  "The server has become ready" means bootstrap, including initial recovery,
  has completed. Scenarios need not repeat initial recovery as a precondition.
  This does not imply that request-driven partition or workspace recovery has
  completed, or that every declared partition was deployed.

  Rule: Initial recovery

    The runtime-recovery scenarios under Partition recovery orchestration,
    Partition checkpoint, Workspace checkpoint, Empty WLog, and Workspace recovery
    orchestration apply to deployed partitions. Commands to undeployed partitions
    do not enter that lifecycle.

    Scenario: Clients can use every declared partition when checkpoints are missing
      Given an application declares partition IDs 0 through 2
      And partition 0 has a stored partition checkpoint
      And partitions 1 and 2 have no stored partition checkpoints
      And each partition is populated with a workspace having this history:
        | partition ID | workspace | last WLog offset | last allocated ordinary record ID |
        | 0            | Orders    | 7                | 200003                            |
        | 1            | Billing   | 14               | 200005                            |
        | 2            | Inventory | 20               | 200009                            |
      And VAOperator has started the VVM
      When a client inserts one ordinary record into each workspace, retrying "503 Service Unavailable" responses
      Then each command eventually returns "200 OK"
      And the successful command responses contain:
        | workspace | CurrentWLogOffset | new ordinary record ID |
        | Orders    | 8                 | 200004                 |
        | Billing   | 15                | 200006                 |
        | Inventory | 21                | 200010                 |

    Scenario Outline: Initial recovery failures are logged while the server becomes ready
      Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
      And initial recovery fails for <failed_partitions> with "PLog read failed"
      When VAOperator starts the VVM
      Then partition checkpoints are <checkpoint_state>
      And initial-recovery error "PLog read failed" is logged for <failed_partitions>
      And the server becomes ready

      Examples:
        | failed_partitions      | checkpoint_state                    |
        | partition 1            | present for partitions 0 and 2 only |
        | partitions 0 through 2 | absent for every partition          |

    Scenario: Commands fail on a failed partition and succeed with the correct sequence numbers on initialized partitions
      Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
      And each partition is populated with a workspace having this history:
        | partition ID | workspace | last WLog offset | last allocated ordinary record ID |
        | 0            | Orders    | 7                | 200003                            |
        | 1            | Billing   | 14               | 200005                            |
        | 2            | Inventory | 20               | 200009                            |
      And initial recovery fails for partition 1 with "PLog read failed"
      And asynchronous projectors and scheduled jobs are configured for every partition
      And VAOperator has started the VVM
      When a client inserts one ordinary record into each workspace, retrying "503 Service Unavailable" responses
      Then the command to Billing returns "500 Internal Server Error"
      And its response error is "partition not found"
      And commands to Orders and Inventory eventually return "200 OK"
      And the successful command responses contain:
        | workspace | CurrentWLogOffset | new ordinary record ID |
        | Orders    | 8                 | 200004                 |
        | Inventory | 21                | 200010                 |
      And partition 1 remains undeployed
      And no asynchronous projector or scheduled job runs for partition 1

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

    Scenario: A later bootstrap deploys a previously failed partition after successful initial recovery
      Given initial recovery of partition 1 of application "test1/app1" failed on the previous VVM start
      And workspace Billing in partition 1 has last WLog offset 14 and last allocated ordinary record ID 200005
      When VAOperator starts the VVM again and initial recovery of partition 1 succeeds
      Then inserting an ordinary record into Billing, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006

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

    Scenario: Requests to a partition being recovered share one recovery
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

    Scenario: The next request reports a failed partition recovery and starts a new attempt
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

  Rule: Partition checkpoint

    usable checkpoint ──────────────────> resume from checkpoint ──┐
    absent or zero checkpoint ──────────> recover from FirstOffset ├─> actual PLog end ──> later command succeeds after workspace recovery
    invalid checkpoint ─> log error ────> recover from FirstOffset ┘

    An absent or zero partition checkpoint starts one scan at FirstOffset.

    Scenario Outline: Partition recovery starts from a checkpoint in basic usage
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

  Rule: Workspace checkpoint

    An absent or zero workspace checkpoint starts one scan at FirstOffset.

    Scenario Outline: Workspace recovery starts from a checkpoint in basic usage
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

    Scenario: A new request to a workspace being recovered does not start another recovery
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

    Scenario: Recovery limit rejects without queueing and permits a later retry
      Given the workspace recovery limit is 4
      And workspaces 42 and 43 are unrecovered after partition recovery
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

    Scenario: A partition reset does not stop ongoing workspace recoveries
      Given the workspace recovery limit is 4
      And workspace 1 is recovered
      And workspace recoveries for workspaces 3, 5, 7, and 9 are running in the same partition
      When a command failure resets and recovers the partition
      And VAOperator submits inserts into unrecovered workspace 11 at these times:
        | request | timing                                          |
        | 1       | before any detached workspace recovery finishes |
        | 2       | after one detached workspace recovery finishes  |
      Then the responses are:
        | request | status                  | reason                        |
        | 1       | 503 Service Unavailable | recovery limit reached        |
        | 2       | 503 Service Unavailable | workspace 11 recovery running |
      And no recovery starts for workspace 11 before request 2
      And request 2 starts exactly one recovery for workspace 11
      And the recoveries for workspaces 3, 5, 7, and 9 complete successfully without restarting

    Scenario: The next request reports a failed workspace recovery and starts a new attempt
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
