Feature: Sequence allocation

  VAOperator issues successful commands and creates workspaces using the next
  offset or identifier within each value's defined scope.

  Rule: Log offsets

    Scenario: PLog offsets advance within an application partition
      Given workspace "Orders" and workspace "Billing" belong to application partition 1
      And the latest PLog event in partition 1 has offset 40
      When VAOperator successfully submits a command in workspace "Orders"
      And VAOperator successfully submits a command in workspace "Billing"
      Then the command in workspace "Orders" is persisted as PLog event 41
      And the command in workspace "Billing" is persisted as PLog event 42

    Scenario: WLog offsets advance independently in each workspace
      Given the WLog of workspace "Orders" ends at offset 7
      And the WLog of workspace "Billing" ends at offset 12
      When VAOperator successfully submits one command in each workspace
      Then the command in workspace "Orders" is persisted at WLog offset 8
      And the command in workspace "Billing" is persisted at WLog offset 13

  Rule: Record IDs

    Scenario: A command receives consecutive ordinary record IDs
      Given the next ordinary record ID in workspace "Orders" is 200001
      When VAOperator successfully creates three ordinary records in one command
      Then the command returns record IDs 200001 through 200003 in creation order
      When VAOperator successfully creates another ordinary record in workspace "Orders"
      Then the command returns record ID 200004

    Scenario: Ordinary record-ID sequences are independent between workspaces
      Given workspace "Orders" and workspace "Billing" have no ordinary records
      When VAOperator successfully creates one ordinary record in each workspace
      Then each command returns record ID 200001

    Scenario: A singleton uses its assigned ID without consuming an ordinary record ID
      Given singleton "Configuration" has application-defined record ID 65536
      And the next ordinary record ID in workspace "Orders" is 200001
      When VAOperator successfully creates the "Configuration" singleton
      Then the stored singleton has record ID 65536
      And the command returns no generated ordinary record ID
      When VAOperator successfully creates an ordinary record in workspace "Orders"
      Then the command returns record ID 200001

  Rule: Workspace IDs

    Scenario: Workspace-ID sequences are scoped to an application
      Given application "Sales" and application "Support" have no user workspaces
      When VAOperator creates workspaces "Orders" and "Billing" in application "Sales"
      And VAOperator creates workspace "Helpdesk" in application "Support"
      Then workspace "Orders" has base WSID 131071
      And workspace "Billing" has base WSID 131072
      And workspace "Helpdesk" has base WSID 131071
