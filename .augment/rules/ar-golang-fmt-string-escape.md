---
type: "always_apply"
---

if you embed string data into a larger string using `fmt.Sprintf`, `fmt.Fprintf`, `fmt.Errorf` or any other `fmt.*` formatter then follow these rules:

## Decision by embedding context

- JSON value (request body, response body, payload field): do not interpolate with `"%s"`; build the structure with `json.Marshal` of a typed struct or `map[string]any`. If a string template must be kept, every string value MUST use `%q` (never `"%s"`)
- URL path segment containing user data: `%s` with `url.PathEscape(seg)`
- URL query parameter value: `%s` with `url.QueryEscape(val)`
- URL host:port: do not build with `fmt.Sprintf("%s:%d", ...)` for user-supplied hosts; use `net.JoinHostPort` (also satisfies `nosprintfhostport`)
- File path: `filepath.Join`, never `fmt.Sprintf("%s/%s", ...)`
- Shell argument: do not interpolate; pass each argument separately to `exec.Command(name, arg1, arg2, ...)`
- SQL literal: do not interpolate; use parameterized queries / placeholders
- HTML attribute or text node: `html.EscapeString(s)` then `%s`, never raw `%s`
- Go-quoted literal embedded into a larger string (single line that should show surrounding quotes): `%q`, never `"%s"`
- Human-readable log / error message where the operand is a free-form string (name, path, error, user input): `%q` over `'%s'` so empty strings and embedded quotes remain visible

## Rules

- never wrap `%s` in double quotes in a format string (`"%s"`); the gocritic check `sprintfQuotedString` enforces this -- use `%q` instead, which adds the quotes and escapes the content
- never wrap `%s` in single quotes in a format string (`'%s'`) for human messages; use `%q` instead
- when the operand is already a `string` or a `Stringer` whose `String()` is known to be safe (e.g. `appdef.QName.String()`, `appdef.AppQName.String()`, integer-derived ids), `%s` without escaping is allowed
- `fmt.Sprintf("%s", x.String())` and `fmt.Sprint(stringer)` are redundant -- use `x.String()` directly (gocritic `redundantSprint`)
- `fmt.Sprintf("%d", i)` where `i` is `int` / `int64` -- use `strconv.Itoa(i)` / `strconv.FormatInt(i, 10)` (perfsprint `integer-format`)
- string concatenation in a loop (`s += x`) -- use `strings.Builder` (perfsprint `concat-loop`)
- when an existing call site builds a JSON body via `fmt.Sprintf` template, prefer rewriting to `json.Marshal`; if the rewrite is out of scope of the current change, at minimum replace every `"%s"` with `%q`

## Anti-patterns

bad -- JSON body with raw `"%s"` (breaks on quotes, backslashes, newlines in the value):

```go
body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":%d}}`, app.Name, app.NumParts)
```

good -- `%q` produces a properly escaped JSON string:

```go
body := fmt.Sprintf(`{"args":{"AppQName":%q,"NumPartitions":%d}}`, app.Name, app.NumParts)
```

better -- `json.Marshal` of a typed value:

```go
b, _ := json.Marshal(map[string]any{"args": map[string]any{"AppQName": app.Name, "NumPartitions": app.NumParts}})
body := string(b)
```

bad -- URL with raw user-supplied query value:

```go
url := fmt.Sprintf("/search?q=%s", q)
```

good:

```go
url := "/search?q=" + url.QueryEscape(q)
```

bad -- redundant `Sprint` of a `Stringer`:

```go
s := fmt.Sprint(qname)
```

good:

```go
s := qname.String()
```

## Checklist for an AI agent before writing any `fmt.Sprintf`/`fmt.Errorf`/`fmt.Fprintf`

1. Classify the embedding context (JSON, URL path, URL query, file path, shell, SQL, HTML, human message, Go-quoted literal)
2. Pick the verb and helper from "Decision by embedding context" above
3. Verify the format string contains no `"%s"` and no `'%s'`
4. If you wrote `fmt.Sprintf("%s", x)` or `fmt.Sprint(x)`, drop the wrapper
5. If the operand is an `int*`, use `strconv` instead of `fmt.Sprintf("%d", ...)`
