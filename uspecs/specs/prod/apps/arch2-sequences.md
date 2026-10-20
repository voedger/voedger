# Sequence architecture: Historical proposal

Current behavior is documented in focused specifications:

- Sequence allocation: [functional design](./sequences.feature) and [technical design](./sequences--td.md)
- Partition and workspace recovery: [functional design](./recovery.feature) and [technical design](./recovery--td.md)

This page preserves the entry point to the former redesign proposal. Its proposed command-processor integration and implementation sketches are historical material; use the focused specifications above for behavior present in the source code.

## Historical context

Both superseded documents remain available at their last modifying commit,
`4899083997529915bf8397e5df9d7f07594da913`:

- [Previous sequence architecture](https://github.com/voedger/voedger/blob/4899083997529915bf8397e5df9d7f07594da913/uspecs/specs/prod/apps/arch-sequences.md)
- [Previous sequence redesign proposal](https://github.com/voedger/voedger/blob/4899083997529915bf8397e5df9d7f07594da913/uspecs/specs/prod/apps/arch2-sequences.md)

The archived documents retain the design alternatives, earlier record-ID ranges, proposed APIs, and historical performance measurements.
