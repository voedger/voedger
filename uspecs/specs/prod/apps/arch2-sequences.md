# Sequence architecture: Historical proposal

Current record-ID and WSID allocation, log-offset bookkeeping, recovery, and storage policy are documented in the [canonical sequence architecture](./arch-sequences.md).

This page preserves the entry point to the former redesign proposal. Its proposed command-processor integration and implementation sketches are historical material; use the canonical document for behavior present in the source code.

## Historical context

[Read the complete previous proposal at its last modifying commit](https://github.com/voedger/voedger/blob/4899083997529915bf8397e5df9d7f07594da913/uspecs/specs/prod/apps/arch2-sequences.md), `4899083997529915bf8397e5df9d7f07594da913`.

The archived version retains the design alternatives, earlier record-ID ranges, proposed APIs, and historical performance measurements. Related snapshots are linked from the [canonical document's historical context](./arch-sequences.md#historical-context).
