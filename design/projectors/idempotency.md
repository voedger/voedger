### Terms
- Idempotent view
  - The view which can not be *updated* twice with the same event

### Motivation
- As a developer, I want that my views are idempotent out of the box so that I shouldn't take care of this in my extensions

### Principles
- All views are idempotent by design. 
- Idempotency is only checked for *Update* intents of the extensions
- No need to check it for *Insert* operations: there's always CanExist / MustNotExist check before Insert operation in the extensions

### Implementation
- An additional field is added by core to store last applied offset to all views
- Existing views:
  - views which does not have this field so once added it will store zero for existing values. Engine should take this into account.
  - view having "offs" field already: check that WLog offset is stored
  - should be possible to rename safely field name from "offs" to some different name
- StateStorage uses this field:
  - To automatically handle the idempotency on every *update* operation
    - Update isn't applied when previous value is already updated with same event in before
    - StateStorage must know the WLog offset of the current event
  - To automatically fill the offset value for *create* operations
  - To send N10n then this field is updated
- Notes regarding the WasmEngine:
  - WasmEngine receives number of intents from ProxyWasm, each has key and value. Those intents which have last applied offset same or higher than the current WLog offset, are dropped
  - Those intents which have no offset filled, are provided with the current WLog offset

### Current Solution
- Developer manually controls the idempotency. 
- N10n is sent by StateStorage when the field "offs" is updated. 
- The same field "offs" is used to control idempotency and sending N10n

