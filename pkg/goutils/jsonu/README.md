# Package jsonu

Package jsonu provides JSON-safe fmt.Sprintf and fmt.Fprintf variants
that JSON-escape string, ~string, fmt.Stringer and error arguments so
callers can build JSON snippets from readable templates.

## Problem

fmt.Sprintf with %q emits Go-quoted strings that are not always valid
JSON (e.g. \v, \xNN), and json.Marshal forces per-value boilerplate to
extract the escaped content from the marshaled bytes.

<details>
<summary>Without jsonu</summary>

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

func main() {
	name := "He said \"hi\"\v"
	err := errors.New("disk\tfull")

	// pitfall: %q emits \v which is not valid JSON
	bad := fmt.Sprintf(`{"name":%q}`, name)
	fmt.Println(json.Valid([]byte(bad))) // false

	// boilerplate: marshal each value and strip the outer quotes
	nb, _ := json.Marshal(name)
	eb, _ := json.Marshal(err.Error())
	body := fmt.Sprintf(
		`{"name":"%s","err":"%s","count":%d}`,
		string(nb[1:len(nb)-1]),
		string(eb[1:len(eb)-1]),
		3,
	)
	fmt.Println(body)
}
```

</details>

<details>
<summary>With jsonu</summary>

```go
package main

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/jsonu"
)

func main() {
	name := "He said \"hi\"\v"
	err := errors.New("disk\tfull")
	body := jsonu.Jprintf(
		`{"name":%q,"err":%q,"count":%d}`, name, err, 3,
	)
	fmt.Println(body)
}
```

</details>

## Features

- **[Jprintf](impl.go#L45)** - JSON-safe fmt.Sprintf for JSON templates
- **[Jfprintf](impl.go#L50)** - JSON-safe fmt.Fprintf writing to an io.Writer

## Use

See [basic usage test](impl_test.go)
