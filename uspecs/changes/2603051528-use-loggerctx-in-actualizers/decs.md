# Decisions: Use context-aware logging in actualizers

## 2. logCtx placement: DoAsync only vs broader scope (keepReading)

Build the base logCtx (vapp + extension, without wsid) in `init()` on the `asyncActualizer` and store it, then enrich with `wsid` per-event in `DoAsync` (confidence: high).

Rationale: `vapp` and `extension` (projector QName) are known at init time, so n10n trace logs in `keepReading` can already carry them. `wsid` is per-event and must be added in `DoAsync`. Storing the base ctx eliminates repetitive `WithContextAttrs` calls and satisfies "as soon as that data is available".

Alternatives:

- Build logCtx only in `DoAsync` as today (confidence: medium)
  - Simpler; n10n trace logs in `keepReading` remain without attributes

## 3. Triggered CUDs for after-insert/update vs all CUDs for after-execute

Reuse `ProjectorEvent` trigger logic inline: iterate `event.CUDs`, check `prj.Triggers(op, type)` per CUD to emit only triggered ones; emit all CUDs when the projector is after-execute (confidence: high).

Rationale: `ProjectorEvent` in `types.go` already contains per-CUD trigger checks (`Insert`, `Update`, `Activate`, `Deactivate`). Reusing the same checks inside a new `logEventAndCUDs` helper avoids duplicating the trigger predicate. Projector type (execute vs CUD-based) is determined by checking `iProjector.Events()` ops.

Alternatives:

- Extract a `triggeringCUDs(prj, event)` helper that returns a slice (confidence: medium)
  - Cleaner API but adds an allocation; log-only path doesn't need the slice outside logging
- Log all CUDs for all projector types (confidence: low)
  - Simpler but noisy for CUD-based projectors that only care about one record type

## 4. Success/failure log location

Log `msg=success` and `msg=failure` inside `DoAsync`, before returning (confidence: high).

Rationale: `logCtx` (carrying `wsid`, `extension`) is only available inside `DoAsync`. Logging there ensures the structured attributes appear on both success and failure entries. The current error propagation path (`wrapErr` → `handleEvent` → `LogError`) logs with `logCtx` already for failures, but adding an explicit `msg=failure` log in `DoAsync` before returning the error gives a symmetric pair and avoids relying on `handleEvent` for the "failure" message.

Alternatives:

- Keep failure logging only in `handleEvent` (confidence: low)
  - The `handleEvent` path already uses `logCtx` via `errWithCtx`, so failure is logged with attributes, but there is no `msg=failure` entry
- Move all error logging into `DoAsync`, remove `errWithCtx` (confidence: medium)
  - Cleaner separation but requires changing `handleEvent` and the error-propagation contract

## 5. args JSON serialization for event log

Use `coreutils.ObjectToMap` + `json.Marshal` on `event.ArgumentObject()`, guard with `event.ArgumentObject().QName() != appdef.NullQName` (confidence: high).

Rationale: This mirrors the command processor's `logEventAndCUDs` approach exactly. Guarding on `NullQName` skips serialization for CUD-only commands (`sys.CUD`) where there is no argument object, keeping the log compact.

Alternatives:

- Always serialize, emit `args={}` when argument is null (confidence: medium)
  - Uniform output but adds a trivial JSON round-trip and an empty field for CUD events
- Use `event.ArgumentObject().AsString(field)` field by field (confidence: low)
  - Requires knowing field names; not generic

## 6. Extract event logging to `pkg/processors` vs keep separate implementations

Extract the shared event/CUD logging skeleton into `pkg/processors.LogEventAndCUDs(...)` and keep the caller-specific behavior in one small per-CUD callback in command processor and actualizers (confidence: high).

Rationale: The two callers share most of the workflow:

- verbose guard
- event context attrs `woffset`, `poffset`, `evqname`
- args JSON logging
- CUD iteration
- per-CUD context attrs `rectype`, `recid`, `op`
- per-CUD `newfields=%s` logging

The remaining differences fit cleanly into one local callback that returns whether the CUD should be logged and which extra message part should be appended:

- command processor logs all CUDs and appends `oldfields=%s`
- actualizer decides whether a CUD triggered the projector and appends nothing

This removes meaningful duplication while keeping the shared API small and preserving the local ownership of projector-specific and old-record-specific behavior.

Alternatives:

- Extract the whole workflow plus old-record handling and projector decision logic into one parameter-heavy helper (confidence: medium)
  - Shares more code but forces unrelated caller details into one API and weakens readability
- Keep only `CudOpToStringForLog` shared (confidence: low)
  - Simpler but leaves most of the duplicated logging workflow in place
