# Set

Memory-efficient bitmap-based set implementation for uint8-based enums
and constants with fast operations and Go 1.23+ iterator support.

## Problem

Working with sets of uint8-based enums requires either verbose map
operations or inefficient slice manipulations that are error-prone
and memory-intensive.

<details>
<summary>Without set</summary>

```go
// Managing enum sets manually - verbose and error-prone
type Status uint8
const (
    StatusActive Status = iota
    StatusPending
    StatusInactive
)

// Verbose map-based approach
validStatuses := make(map[Status]struct{})
validStatuses[StatusActive] = struct{}{}
validStatuses[StatusPending] = struct{}{}

// Error-prone membership checks
func isValidStatus(s Status) bool {
    _, exists := validStatuses[s]
    return exists // boilerplate here
}

// Complex set operations
func combineStatuses(set1, set2 map[Status]struct{}) map[Status]struct{} {
    result := make(map[Status]struct{})
    for k := range set1 {
        result[k] = struct{}{} // repetitive boilerplate
    }
    for k := range set2 {
        result[k] = struct{}{} // common mistake: forgetting duplicates
    }
    return result
}

// Iteration requires extra steps
var statusList []Status
for status := range validStatuses {
    statusList = append(statusList, status)
}
```

</details>

<details>
<summary>With set</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/set"

// Clean and simple
validStatuses := set.From(StatusActive, StatusPending)

// Simple membership check
isValid := validStatuses.Contains(status)

// Easy set operations
combined := set.Empty[Status]()
combined.Collect(set1.Values())
combined.Collect(set2.Values())

// Direct iteration
for status := range validStatuses.Values() {
    // process status
}
```

</details>

## Features

- **[Bitmap storage](set.go#L19)** - Uses 256-bit bitmap for all uint8
  values with constant memory footprint
  - [Bit manipulation core: set.go#L276](set.go#L276)
  - [Efficient bit counting: set.go#L209](set.go#L209)
  - [Binary serialization: set.go#L84](set.go#L84)
- **[Iterator support](set.go#L52)** - Full Go 1.23+ iterator
  compatibility for modern range loops
  - [Forward iteration: set.go#L260](set.go#L260)
  - [Backward iteration: set.go#L97](set.go#L97)
  - [Indexed iteration: set.go#L52](set.go#L52)
- **[Immutable variants](set.go#L37)** - Read-only sets for
  constants and shared data
  - [Read-only constructor: set.go#L37](set.go#L37)
  - [Immutability enforcement: set.go#L235](set.go#L235)
- **[Batch operations](set.go#L166)** - Efficient collection and
  range operations
  - [Iterator collection: set.go#L166](set.go#L166)
  - [Range setting: set.go#L226](set.go#L226)
  - [Chunked iteration: set.go#L123](set.go#L123)

## Use

See [example](example_test.go)
