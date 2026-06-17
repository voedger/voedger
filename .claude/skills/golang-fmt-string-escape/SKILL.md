---
name: golang-fmt-string-escape
description: Use this skill when writing or reviewing any `fmt.Sprintf`, `fmt.Errorf`, `fmt.Fprintf`, `fmt.Printf` (or other `fmt.*` formatter) call in Go code that embeds string data into a JSON value, URL path/query, host:port, file path, shell argument, SQL literal, HTML attribute/text, Go-quoted literal, or human-readable log/error message. Picks the correct verb (`%s` / `%q`) and escaping helper (`jsonu.Jprintf`, `jsonu.Jfprintf`, `json.Marshal`, `url.PathEscape`, `url.QueryEscape`, `net.JoinHostPort`, `filepath.Join`, `html.EscapeString`, parameterized SQL). Always classify each call site individually by its sink (how the result is used) and the source of every string operand (whether it can contain sink-breaking bytes); never apply a blanket, sink-blind verb swap such as `"%s"` -> `%q`. JSON-string escaping rules do not apply to `_test.go` files.
user-invocable: false
---

## Core principle -- classify every site individually

Before choosing a verb or helper for any `fmt.*` formatter call, answer two questions, per call site and never in bulk:

1. Sink -- how is the result used (JSON value, URL path/query, host:port, file path, shell arg, SQL, HTML, Go-quoted literal, human message)? see the table below
2. Source -- where does each string operand come from, and can it contain bytes that break that sink? user input / external API / DB field / event payload -> assume unsafe; fixed enum / `appdef.QName.String()` / integer-derived id -> safe-ASCII

A blanket, sink-blind transformation is forbidden -- in particular never mechanically rewrite every `"%s"` to `%q` across a file. `%q` is correct only in some sinks (Go-quoted literal, JSON with safe-ASCII operands) and wrong in others (HTML, or JSON with unsafe operands where it can emit `\v` / `\xNN` that JSON parsers reject). Clearing the `sprintfQuotedString` linter is a side effect of the correct fix, not the goal

## Decision by embedding context

| Context                                   | How to interpolate                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| JSON value (non-test)                     | `jsonu.Jprintf` with `"%s"` (content, caller quotes) or `%q` (full JSON literal) -- default. `jsonu.Jfprintf(w, ...)` when writing directly to an `io.Writer` (`http.ResponseWriter`, `*bytes.Buffer`, ...) -- same verbs and escaping as `Jprintf`. `json.Marshal` of a typed value only when there is no template at all. `fmt.Sprintf` with `%q` only for safe-ASCII operands (`appdef.QName.String()`, identifiers, fixed enums) -- Go's `%q` emits `\v` / `\xNN` that JSON parsers reject. Never raw `"%s"` |
| URL path segment                          | `%s` with `url.PathEscape(seg)`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| URL query value                           | `%s` with `url.QueryEscape(val)`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| URL host:port                             | `net.JoinHostPort` (also satisfies `nosprintfhostport`); never `fmt.Sprintf("%s:%d", ...)`                                                                                                                                                                                                                                                                                                                                                                                                                       |
| File path                                 | `filepath.Join`; never `fmt.Sprintf("%s/%s", ...)`                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Shell argument                            | pass each arg to `exec.Command(name, arg1, arg2, ...)`; never interpolate                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| SQL literal                               | parameterized queries / placeholders; never interpolate                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HTML attribute or text node               | `"%s"` with `html.EscapeString(s)`; never raw `%s`, never `%q` (Go-quoting is not HTML escaping; an embedded `"` still breaks out of the attribute)                                                                                                                                                                                                                                                                                                                                                              |
| URL inside HTML attribute (`href`, `src`) | URL-escape the user parts first (`url.PathEscape` / `url.QueryEscape`), then `html.EscapeString` the whole URL, then `"%s"`                                                                                                                                                                                                                                                                                                                                                                                      |
| Go-quoted literal inside a larger string  | `%q`; never `"%s"`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Human log / error message                 | `%q` over `'%s'` so empty strings and embedded quotes stay visible                                                                                                                                                                                                                                                                                                                                                                                                                                               |

`_test.go` exception: in test-only JSON fixtures / request bodies / expected snippets, plain `fmt.Sprintf` with `"%s"` or literal quoted strings is allowed. All other contexts still apply.

## Rules

- never wrap `%s` in double quotes (`"%s"`) in a raw `fmt.Sprintf` outside the `_test.go` JSON exception; the gocritic `sprintfQuotedString` enforces it. For JSON use `jsonu.Jprintf` (keeps `"%s"`) or `%q`; for Go-quoted literals use `%q`; for HTML keep `"%s"` AND switch escaper to `html.EscapeString`
- never wrap `%s` in single quotes (`'%s'`) in a human message; use `%q`
- never `json.Marshal` an individual string just to embed it in a larger JSON template -- use `jsonu.Jprintf` instead
- safe `Stringer` operand (e.g. `appdef.QName.String()`, integer-derived ids): raw `%s` without extra escaping is allowed only for human messages; JSON, HTML and URL still require their context escaper (the risk there is structural, not value sanitization)
- `jsonu.Jprintf` / `json.Marshal` of a Go `string` coerces invalid UTF-8 to U+FFFD; for arbitrary binary bytes marshal a `[]byte` (encoded as base64)
- `fmt.Sprintf("%s", x.String())` / `fmt.Sprint(stringer)` -- use `x.String()` (gocritic `redundantSprint`)
- `fmt.Sprintf("%d", i)` for `int` / `int64` -- use `strconv.Itoa` / `strconv.FormatInt` (perfsprint `integer-format`)
- string concatenation in a loop -- use `strings.Builder` (perfsprint `concat-loop`)
- a JSON template (`fmt.Sprintf` or `jsonu.Jprintf`) MUST be self-balanced: `{...}` and `[...]` in the same template; never split the closing brace into a different branch / write
- if a JSON object MUST be streamed across multiple writes, exactly one piece of code owns the opening `{` and its matching `}` and emits both on every reachable path (success, error, empty)
- ad-hoc error / status responses (`{"status":N,"errorDescription":"..."}`): emit the whole object in one `Write` -- a `jsonu.Jprintf` template (default), a `json.Marshal`ed struct (typed schema), or a `fmt.Sprintf` template with `%q` (safe-ASCII operands)
- when an existing non-test JSON `fmt.Sprintf` site is touched, rewrite it to `jsonu.Jprintf` (or `json.Marshal` of a typed value); the mechanical `"%s"` -> `%q` swap is NOT an acceptable substitute -- it only silences `sprintfQuotedString` while leaving the wrong helper in place for unsafe operands
- streaming JSON to an `io.Writer` (`http.ResponseWriter`, `*bytes.Buffer`, ...) -- use `jsonu.Jfprintf` instead of `fmt.Fprintf`; same verb rules as `jsonu.Jprintf`. The returned `error` is infallible for `*bytes.Buffer`: handle with `// notest` + `panic(err)` per `ar-golang.md`
- automated / AST tooling for these sites may only DETECT and PRINT each call site (file:line, the format string, and the operand expressions) for individual review; it MUST NOT auto-apply a verb swap. A reported site is then classified by sink + source and fixed one at a time

## Anti-patterns

bad -- raw `"%s"` in JSON (breaks on quotes, backslashes, newlines):

```go
body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":%d}}`, app.Name, app.NumParts)
```

bad -- `json.Marshal` an individual string to embed it (verbose, throwaway var, ignored error):

```go
name, _ := json.Marshal(app.Name)
body := fmt.Sprintf(`{"args":{"AppQName":%s,"NumPartitions":%d}}`, name, app.NumParts)
```

good -- `jsonu.Jprintf` (default for JSON construction):

```go
body := jsonu.Jprintf(`{"args":{"AppQName":"%s","NumPartitions":%d}}`, app.Name, app.NumParts)
```

acceptable -- `fmt.Sprintf` with `%q` for safe-ASCII operands only:

```go
body := fmt.Sprintf(`{"args":{"AppQName":%q,"NumPartitions":%d}}`, app.Name, app.NumParts)
```

acceptable -- `json.Marshal` of a typed value when there is no template:

```go
b, _ := json.Marshal(map[string]any{"args": map[string]any{"AppQName": app.Name, "NumPartitions": app.NumParts}})
body := string(b)
```

bad / good -- URL query value:

```go
url := fmt.Sprintf("/search?q=%s", q)     // bad
url := "/search?q=" + url.QueryEscape(q)  // good
```

bad -- HTML attribute with `%q` (Go-quoting is not HTML escaping):

```go
html := fmt.Sprintf(`<a href=%q>%s</a>`, ref, text)
```

good -- `"%s"` with `html.EscapeString` for both attribute and text:

```go
html := fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(ref), html.EscapeString(text))
```

good -- URL inside `href`: URL-escape the user parts first, then HTML-escape the whole URL:

```go
ref := "/items/" + url.PathEscape(itemID) + "?q=" + url.QueryEscape(query)
html := fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(ref), html.EscapeString(text))
```

bad / good -- redundant `Sprint` of a `Stringer`:

```go
s := fmt.Sprint(qname)  // bad
s := qname.String()     // good
```

## Checklist before writing or changing any `fmt.Sprintf` / `fmt.Errorf` / `fmt.Fprintf`

1. Identify the sink (embedding context) -- see the table above
2. Identify the source of every string operand and decide whether it can contain sink-breaking bytes -- default to unsafe unless it is a fixed enum / `Stringer` id
3. `_test.go` JSON fixtures: plain `"%s"` allowed; all other contexts still apply
4. Pick the verb and escaper from the table for that exact sink + source -- one site at a time, never a bulk verb swap
5. A `sprintfQuotedString` linter hit is a prompt to classify the site (steps 1-2), not a license to blanket-replace `"%s"` with `%q`; the linter going green does not prove the helper is correct
6. Drop `fmt.Sprintf("%s", x)` / `fmt.Sprint(x)`; use `strconv` instead of `fmt.Sprintf("%d", ...)`
