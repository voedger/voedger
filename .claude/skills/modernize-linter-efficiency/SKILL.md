---
name: modernize-linter-efficiency
description: Decide whether to accept or suppress a Go `modernize` or `exptostd` linter suggestion based on measured CPU and memory efficiency. Use when reviewing or applying these linters' rewrites (e.g. strings.SplitSeq, maps.Collect, slices.ContainsFunc, slices.Values, materializing golang.org/x/exp/maps.Keys / maps.Values into slices.AppendSeq/slices.Collect), or when deciding whether a `//nolint` suppression is justified. Backed by reproducible benchmarks in air-rsch
---

# Modernize / exptostd efficiency decisions

`modernize` and `exptostd` suggest newer stdlib constructs. Most are wins, but a few trade CPU or allocations for readability. This skill encodes which to accept and which to suppress, with benchmark evidence.

## Default rule

Accept the suggested modern form. Reject it only when a benchmark shows a **>5% regression in CPU time or in allocations** (bytes or count) at a realistic input size. When the modern form shows **no measurable benefit**, prefer the shorter, clearer code — readability is the tiebreaker.

When unsure, measure: write paired `BenchmarkOld`/`BenchmarkNew` tests with `b.ReportAllocs()`, run with `-count=10`, and compare with `benchstat`. See [results.md](results.md) for the captured evidence.

## Per-rule verdicts

Evidence: [results.md](results.md).

| Rule                                 | Construct                                                                   | Verdict                                                                                                                                                                                                                                                                                                                                                                       |
|--------------------------------------|-----------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `stringsseq`                         | `strings.Split` → `strings.SplitSeq`                                        | **Accept** — ~60–70% faster, zero allocations                                                                                                                                                                                                                                                                                                                                 |
| `exptostd` (maps.Keys / maps.Values) | `x/exp/maps.Keys(m)` → `slices.AppendSeq(make([]T,0,len(m)), maps.Keys(m))` | **Reject for `Keys`/`Values` only (suppressed via `issues.exclude-rules` in `.golangci.yml`)** — performance and allocations are identical, but the rewrite is verbose and reads worse than the one-line `x/exp` call. The `x/exp` dependency is cheap to keep; readability wins. Other `exptostd` rewrites (`Clone`, `Equal`, `DeleteFunc`, etc.) stay enabled — accept them |
| `slicescontains`                     | `==` loop → `slices.Contains`; predicate loop → `slices.ContainsFunc`       | **Accept by default; keep loop in hot paths** — zero extra allocations, but both helpers add ~65–100% CPU on late/absent scans of a large slice                                                                                                                                                                                                                               |
| `mapsloop`                           | pre-sized loop → `maps.Collect(seq)`                                        | **Reject** — `maps.Collect` can't pre-size from `iter.Seq2`; +80–130% CPU and +150–175% allocations                                                                                                                                                                                                                                                                           |
| range vs iterator                    | `for range s` → `for range slices.Values(s)`                                | **Keep direct range** — statistically equivalent; use `slices.Values` only when an `iter.Seq` is genuinely needed                                                                                                                                                                                                                                                             |

### Notes

- `stringsseq`: the old form allocates a `[]string`; `SplitSeq` streams substrings with no backing slice. Always take it.
- `exptostd`: the rule's only purpose is dropping `golang.org/x/exp/maps`. Allocations and CPU match. But the `Keys`/`Values` rewrites turn a single-token `maps.Keys(m)` into `slices.AppendSeq(make([]T,0,len(m)), maps.Keys(m))` — three nested calls, an explicit element type, and the same map referenced twice. Readability cost is real; dependency cost is negligible (the package is one file, no transitive deps). The linter stays enabled; only the `Keys`/`Values` messages (and the consequent import warning) are suppressed via `issues.exclude-rules` in `.golangci.yml`. Do NOT rewrite `Keys`/`Values` call sites manually. Other `exptostd` suggestions (e.g. `maps.Clone`, `maps.Equal`, `maps.DeleteFunc`) are still active — accept them.
- `slicescontains`: both `slices.Contains` (equality) and `slices.ContainsFunc` (predicate) allocate nothing, but on a late/absent scan of a large slice they cost ~1.6–2× the value-range hand loop (bounds-check + closure indirection). Prefer the helper for readability; keep the loop on hot, large-slice scans.
- `mapsloop`: the clearest reject. Keep `m := make(map[K]V, size)` followed by a copy loop when you already know the size.
- range/iterator: a tie, so keep the simpler direct range. Reach for `slices.Values` only to compose with iterator APIs.

## Exception protocol

When you keep an old construct against a linter suggestion, suppress the warning at the flagged line and cite the measured regression:

```go
//nolint:modernize // maps.Collect regresses +131% CPU / +175% allocs vs pre-sized loop; see results.md
```

```go
//nolint:modernize // hot scan: slices.ContainsFunc regresses ~95–100% CPU on late/absent vs hand loop; see results.md
```

For `exptostd` the suppression is repo-wide in `.golangci.yml`, not per-line — the verdict applies to every `Keys`/`Values` call site equally. The linter stays enabled; only the two message patterns (plus the consequent import warning) are excluded:

```yaml
issues:
  exclusions:
    rules:
      - linters:
          - exptostd
        # maps.Keys/Values rewrites hurt readability; the import-statement message is the consequence
        text: golang\.org/x/exp/maps\.(Keys|Values)|Import statement.+golang\.org/x/exp/maps
```

Rules for a valid suppression:

- Name the specific linter and rule
- State the measured regression (CPU and/or allocation percentage) at a realistic size
- Link to `results.md` (or a fresh benchmark) as evidence

Do not suppress merely because the modern form is unfamiliar — only a measured >5% regression justifies it.

## Reproducing

```sh
cd modernize-bench
go test -run=^$ -bench=. -benchmem -count=10 | tee bench.txt
benchstat bench.txt   # or split into old.txt/new.txt for an A/B view
```

If a future Go release closes one of these gaps (especially `mapsloop` or `slicescontains`), re-run and update the verdict table and `results.md`.
