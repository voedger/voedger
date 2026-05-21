---
name: golang-fmt-string-escape
description: Use this skill when writing or reviewing any `fmt.Sprintf`, `fmt.Errorf`, `fmt.Fprintf`, `fmt.Printf` (or other `fmt.*` formatter) call in Go code that embeds string data into a JSON value, URL path/query, host:port, file path, shell argument, SQL literal, HTML attribute/text, Go-quoted literal, or human-readable log/error message. Picks the correct verb (`%s` / `%q`) and escaping helper (`jsonu.Jprintf`, `json.Marshal`, `url.PathEscape`, `url.QueryEscape`, `net.JoinHostPort`, `filepath.Join`, `html.EscapeString`, parameterized SQL). JSON-string escaping rules do not apply to `_test.go` files.
user-invocable: false
---

## Decision by embedding context

| Context                                   | How to interpolate                                                                                                                                                                                                                                                                                                                                       |
| ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| JSON value (non-test)                     | `jsonu.Jprintf` with `"%s"` (content, caller quotes) or `%q` (full JSON literal) -- default. `json.Marshal` of a typed value only when there is no template at all. `fmt.Sprintf` with `%q` only for safe-ASCII operands (`appdef.QName.String()`, identifiers, fixed enums) -- Go's `%q` emits `\v` / `\xNN` that JSON parsers reject. Never raw `"%s"` |
| URL path segment                          | `%s` with `url.PathEscape(seg)`                                                                                                                                                                                                                                                                                                                          |
| URL query value                           | `%s` with `url.QueryEscape(val)`                                                                                                                                                                                                                                                                                                                         |
| URL host:port                             | `net.JoinHostPort` (also satisfies `nosprintfhostport`); never `fmt.Sprintf("%s:%d", ...)`                                                                                                                                                                                                                                                               |
| File path                                 | `filepath.Join`; never `fmt.Sprintf("%s/%s", ...)`                                                                                                                                                                                                                                                                                                       |
| Shell argument                            | pass each arg to `exec.Command(name, arg1, arg2, ...)`; never interpolate                                                                                                                                                                                                                                                                                |
| SQL literal                               | parameterized queries / placeholders; never interpolate                                                                                                                                                                                                                                                                                                  |
| HTML attribute or text node               | `"%s"` with `html.EscapeString(s)`; never raw `%s`, never `%q` (Go-quoting is not HTML escaping; an embedded `"` still breaks out of the attribute)                                                                                                                                                                                                      |
| URL inside HTML attribute (`href`, `src`) | URL-escape the user parts first (`url.PathEscape` / `url.QueryEscape`), then `html.EscapeString` the whole URL, then `"%s"`                                                                                                                                                                                                                              |
| Go-quoted literal inside a larger string  | `%q`; never `"%s"`                                                                                                                                                                                                                                                                                                                                       |
| Human log / error message                 | `%q` over `'%s'` so empty strings and embedded quotes stay visible                                                                                                                                                                                                                                                                                       |

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
- when an existing non-test JSON `fmt.Sprintf` site is being touched, rewrite to `jsonu.Jprintf`; if a rewrite is out of scope, at minimum replace every `"%s"` with `%q`

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

## Checklist before writing any `fmt.Sprintf` / `fmt.Errorf` / `fmt.Fprintf`

1. Classify the embedding context (see table above)
2. `_test.go` JSON fixtures: plain `"%s"` allowed; all other contexts still apply
3. Pick the verb and escaper from the table for that exact context
4. On a `sprintfQuotedString` linter hit -- in JSON switch to `jsonu.Jprintf` (keep `"%s"`) or `%q` (safe-ASCII only); in HTML switch the escaper to `html.EscapeString` and keep `"%s"` (do NOT switch to `%q`)
5. Drop `fmt.Sprintf("%s", x)` / `fmt.Sprint(x)`; use `strconv` instead of `fmt.Sprintf("%d", ...)`
