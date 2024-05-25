# [require](https://pkg.go.dev/github.com/voedger/voedger/pkg/goutils/testingu/require) package

The package `require` in addition to the Testify [require](https://pkg.go.dev/github.com/stretchr/testify/require) package provides a set of functions for checking panic and errors.

## Check panic

- Check that the object recovered from panic contains (or does not contains) the specified substrings
- Check that the object recovered from panic matches (or does not matches) by the specified regular expression
- Check that the error recovered from panic (or err's chain) is the target error

### Example 1

This example demonstrates how to test that the function causes panic with the expected error and a message containing the expected substrings.

```go
package yours_test

import (
  "errors"
  "fmt"
  "testing"

  "github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestPanics(t *testing.T) {
  require := require.New(t)
  
  myError := fmt.Errorf("my error: %w", errors.ErrUnsupported)

  require.PanicsWith(
    func() {    panic(myError)  },
    require.Is(myError, "panic error should be %v", myError),
    require.Is(errors.ErrUnsupported),
    require.Has("my"),
    require.Has("unsupported"),
  )
}
```

## Check error

- Check that the error contains (or does not contains) the specified substring
- Check that the error matches (or does not matches) by the specified regular expression
- Check that an error (or err's chain) is the target error

### Example 2

This example demonstrates how to test that the error is expected target error, contains the expected substrings and matches by the regular expression.

```go
package yours_test

import (
  "errors"
  "fmt"
  "testing"

  "github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestErrors(t *testing.T) {
  require := require.New(t)
  
  myError := fmt.Errorf("my error: %w", errors.ErrUnsupported)

  require.ErrorWith(
    fmt.Errorf("boom: %w", myError),
    require.Is(myError),
    require.Is(errors.ErrUnsupported, "tested error should be %v", errors.ErrUnsupported),
    require.Has("boom"),
    require.Has("my"),
    require.Has("unsupported"),
    require.Rx(".*:.*:.*", "error should contains two `:`", ".*:.*:.*"),
  )
}
```
