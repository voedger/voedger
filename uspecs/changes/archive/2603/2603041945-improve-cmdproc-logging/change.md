---
registered_at: 2026-03-04T18:19:40Z
change_id: 2603041819-improve-cmdproc-logging
baseline: ae8c403a67439e9cbc987fe7693b7879c561b495
archived_at: 2026-03-04T19:45:16Z
---

# Change request: Improve logging in command processor

## Why

Command processor currently lacks structured log entries for events and CUD (Create/Update/Delete) operations, making it difficult to trace what happened before an event is persisted to the plog.

## What

Add two structured log entries emitted before an event is saved to plog:

Event log entry:

- ctx attributes: woffset, poffset
- msg: args (json form)

CUD log entry (one per CUD record):

- msg: (implicit per-record entry)
	- rectype: e.g. `untill.cdoc.article`
	- recid: actual ID (not raw ID), e.g. `78097`
	- op: `insert`, `update`, etc.
	- newfields: string representation of new field values
	- oldfields: string representation of old field values
