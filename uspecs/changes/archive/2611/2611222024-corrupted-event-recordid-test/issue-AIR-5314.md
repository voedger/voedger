# voedger: cover by tests the problematic case when impossible to recover RecorID after corrupted event

- URL: https://untill.atlassian.net/browse/AIR-5314
- ID: AIR-5314
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-4959 (parent: voedger: partition recovery performance)

## Why

Case:

* ID-bearing wlog event is marked as corrupt
* there are no ID-bearing events after it

Impossible to determine the next RecordID because we do not know the last issued RecordID from the corrupted event

## What

* Cover by gherkin scenario this case. Given Trust level is “do not trust Put Record”. Observable consequence: server 500 with `sequences violation` message on simple inserting a record after corrupting the event
* implement a corresponding integration test that expects that error

