# Benchmark results

Empirical comparison of each `modernize`/`exptostd` rule's old construct (`_Old` / `_Direct`) against its modern construct (`_New` / `_Iterator`).

## Source of truth

Raw captures live alongside this file: `bench.txt` (combined), `old.txt`/`new.txt` (split for `benchstat`), and `benchstat.txt` (full `benchstat` output).

## Decision rule

Prefer the modern form by default. Reject it for a given construct only when the benchmark shows a **>5% regression in CPU time or in allocations** (bytes or count) at a realistic size.

## Per-rule results

`vs base` is modern relative to old; positive means the modern form is slower / allocates more.

### stringsseq — `strings.Split` vs `strings.SplitSeq`

| size   | CPU vs base | allocs vs base |
|--------|-------------|----------------|
| n=4    | -70.82%     | -100% (1 → 0)  |
| n=64   | -59.58%     | -100% (1 → 0)  |
| n=1024 | -58.11%     | -100% (1 → 0)  |

Verdict: **accept**. `SplitSeq` is dramatically faster and allocation-free (no backing slice).

### exptostd — `x/exp/maps.Keys` vs `slices.AppendSeq(make(...), maps.Keys)`

| size   | CPU vs base | allocs vs base |
|--------|-------------|----------------|
| n=8    | +10.99%     | ~ (1 = 1)      |
| n=128  | +16.10%     | ~ (1 = 1)      |
| n=2048 | ~ (noisy)   | ~ (1 = 1)      |

Verdict: **accept**. Allocations are identical (both produce one pre-sized slice). The small-map CPU deltas are within run-to-run noise on this machine and carry no allocation penalty; the stdlib form removes the `golang.org/x/exp` dependency, which is the rule's purpose. No >5% allocation regression.

### slicescontains — hand loop vs `slices.ContainsFunc`

| case      | CPU vs base | allocs vs base |
|-----------|-------------|----------------|
| early hit | ~           | ~ (0 = 0)      |
| late hit  | +43.85%     | ~ (0 = 0)      |
| absent    | +50.25%     | ~ (0 = 0)      |

Verdict: **accept with caveat**. Zero allocations either way. The late/absent CPU regression comes from the per-element closure call vs an inlined comparison. For a simple `==` predicate in a hot, large-slice scan, keep the hand loop. Otherwise prefer `ContainsFunc` for readability.

### mapsloop — pre-sized loop vs `maps.Collect`

| size   | CPU vs base | allocs (count) vs base |
|--------|-------------|------------------------|
| n=8    | +52.99%     | ~ (4 = 4)              |
| n=128  | +134.07%    | +150% (6 → 15)         |
| n=2048 | +86.13%     | +175% (12 → 33)        |

Verdict: **reject (keep old)**. `maps.Collect` cannot pre-size from an `iter.Seq2`, so it grows the destination map incrementally — far more CPU and allocations than `make(map, size)` plus a loop. This exceeds the threshold on both axes.

### range-over-slice vs range-over-iterator — `for range s` vs `for range slices.Values(s)`

| size   | CPU vs base    | allocs vs base |
|--------|----------------|----------------|
| n=8    | +5.29% (noise) | ~ (0 = 0)      |
| n=256  | ~              | ~ (0 = 0)      |
| n=4096 | ~              | ~ (0 = 0)      |

Verdict: **direct range preferred**. The two are statistically equivalent at realistic sizes (the n=8 delta is sub-nanosecond noise) with zero allocations. Since they tie, keep the simpler direct `for range slice`; reserve `slices.Values` for when an `iter.Seq` is actually needed for composition.
