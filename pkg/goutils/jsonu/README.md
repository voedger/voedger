# Package jsonu

Package jsonu formats JSON snippets that still use fmt-style
templates. It JSON-escapes string-like arguments (`string`, named
`~string` types, and `fmt.Stringer` implementations) while leaving
other values to normal fmt formatting.

## Problem

JSON string values need JSON escaping, but fmt `%q` uses Go string
escaping and can emit sequences JSON parsers reject. This package keeps
small JSON templates readable while avoiding malformed output.

<details>
<summary>Without jsonu</summary>

```go
package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	verticalTab := "line\vbreak"
	badVTab := fmt.Sprintf(`{"name":%q}`, verticalTab)
	fmt.Println(json.Valid([]byte(badVTab)))
	// false: Go \v escape is not valid JSON

	invalidUTF8 := string([]byte{0xff})
	badByte := fmt.Sprintf(`{"name":%q}`, invalidUTF8)
	fmt.Println(json.Valid([]byte(badByte)))
	// false: Go \xNN escape is not valid JSON

	name := "line\vwith \"quotes\""
	nameJSON, err := json.Marshal(name)
	if err != nil {
		panic(err)
	}

	// boilerplate for every string inserted into the template
	nameEscaped := string(nameJSON[1 : len(nameJSON)-1])
	body := fmt.Sprintf(
		`{"name":"%s","count":%d}`,
		nameEscaped,
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
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/jsonu"
)

type qName string

func (q qName) String() string { return string(q) }

func main() {
	body := jsonu.Jprintf(
		`{"qname":%q,"name":%q,"count":%d}`,
		qName(`app.Doc`),
		"line\vwith \"quotes\"",
		3,
	)
	fmt.Println(body)
}
```

</details>

## Features

- **[Jprintf](impl.go)** - Formats JSON string templates safely
  - Escapes `string`, named `~string` types, and `fmt.Stringer` arguments
  - Forwards all other arguments to `fmt.Sprintf` unchanged (`%d`, `%t`, `%g`, ...)
  - `%s` and `%v` emit escaped content; the caller supplies the surrounding
    quotes in the template
  - `%q` emits a complete JSON string literal (escaped content wrapped in
    double quotes); no quotes are needed in the template
  - Honors flags and width on string-like arguments (e.g. `%-10.3s`, `%10q`)

## Use

```go
name := "line\vwith \"quotes\""

// %s: provide the quotes in the template
body := jsonu.Jprintf(`{"name":"%s","count":%d}`, name, 3)

// %q: equivalent, quotes are produced by Jprintf
body = jsonu.Jprintf(`{"name":%q,"count":%d}`, name, 3)
```
