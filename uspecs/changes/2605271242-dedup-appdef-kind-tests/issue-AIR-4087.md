# voedger: consolidate copy-paste tests in pkg/appdef: enum kinds and doc kinds

- URL: https://untill.atlassian.net/browse/AIR-4087
- ID: AIR-4087
- State: in-progress
- Author: d.gribanov@dev.untill.com
- Assignees: d.gribanov@dev.untill.com
- Labels: none

## Description

Why
The pkg/appdef test suite contains two clusters of structurally identical, copy-pasted tests that grow linearly with every new kind added:
Seven test files cover MarshalText / TrimString for ~uint8 enums (ConstraintKind, DataKind, ExtensionEngineKind, FilterKind, TypeKind, LimitFilterOption, RateScope) — each repeats the same table-driven 15-line pattern
Four test files ( cdoc_test.go,gdoc_test.go,odoc_test.go,wdoc_test.go) are 83-line twins differing only in ~10 symbols (builder method, lookup function, iterator, expected TypeKind)
Consequences:
Adding a new enum kind or doc kind requires copy-pasting an entire file and renaming symbols, with no compile-time guarantee that the new kind gets the same coverage as existing ones
Fixing a defect in the shared logic requires N identical edits, with high risk of forgetting one
~580 LOC of repetition obscures the actual test intent and inflates review surface for every kind-related change

What
Consolidate the duplicated tests into two fixture-driven scenarios under pkg/appdef:
One unified scenario for enum kinds (MarshalText and TrimString) parameterised by the enum type, covering all ~uint8 kinds currently tested individually
One unified scenario for doc kinds (CDoc, GDoc, ODoc, WDoc) parameterised by builder + lookup + enumerator callbacks, covering build, find-by-name, container-kind, nil-on-unknown, and enumeration
