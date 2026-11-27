# voedger: I*LogEvent.Release() missing in few places

- URL: https://untill.atlassian.net/browse/AIR-5377
- ID: AIR-5377
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignee: Denis Gribanov
- Linked issue: [AOPS-55: Research partition recovery throughput bottleneck](https://untill.atlassian.net/browse/AOPS-55)

## Why

Several readers finish using an `IPLogEvent` or `IWLogEvent` without calling `Release()`. A SQL log scan skips cleanup for every event it reads. An async actualizer can also abandon part of a batch when a send fails or processing is canceled.

Skipped releases prevent event buffers from returning to their pools and leave reader references unbalanced on PLog cache hits. Repeated scans can increase allocation and garbage collection costs.

## What

* Release each event after SQL log rendering and after `GetORec` finishes searching or copying the requested record
* In `updateCorrupted`, retain the source event until the replacement-event builder has copied `event.Bytes()`, then release it
* Define who releases events returned by `wLogStorage.Get` and `Read`, so journal consumers and retained state values remain valid through their last use
* On async actualizer send failure or cancellation, release every event in the batch that was never handed to the pipeline
* Return pooled resources when a single-event PLog or WLog read fails during decoding, before the callback receives the event
* Update affected tests and verify that each acquired event is released exactly once across successful reads, callback errors, send failures, and cancellation
