# Package jsonu

Package jsonu formats JSON snippets that still use fmt-style
templates. It JSON-escapes string-like arguments while leaving other
values to normal fmt formatting.

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

- **[Jprintf](impl.go#L22)** - Formats JSON string templates safely

## Use

```go
name := "line\vwith \"quotes\""
body := jsonu.Jprintf(
	`{"name":"%s","count":%d}`, // %q could be used as well
	name,
	3,
)
```
