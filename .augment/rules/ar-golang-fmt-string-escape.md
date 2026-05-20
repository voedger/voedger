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
- HTML attribute or text node: `html.EscapeString(s)` then `"%s"`, never raw `%s`, and never `%q` (Go-quoting `\"`, `\n` is not interpreted by HTML parsers and a quote in the operand can still break out of the attribute)
- URL embedded into an HTML attribute (e.g. `href`, `src`): URL-escape the user-controlled parts first (`url.PathEscape` / `url.QueryEscape`), then `html.EscapeString` the whole URL, then `"%s"`
- Go-quoted literal embedded into a larger string (single line that should show surrounding quotes): `%q`, never `"%s"`
- Human-readable log / error message where the operand is a free-form string (name, path, error, user input): `%q` over `'%s'` so empty strings and embedded quotes remain visible

## Rules

- classify the embedding context first (see "Decision by embedding context"); the `%s` / `%q` choice is context-dependent and the rules below apply only inside their stated context
- in JSON values and Go-quoted literals: never wrap `%s` in double quotes in a format string (`"%s"`); the gocritic check `sprintfQuotedString` enforces this -- use `%q` instead, which adds the quotes and escapes the content per Go-string rules
- in HTML attributes and text nodes: `%q` is WRONG (Go-string escaping, not HTML escaping); keep the literal double quotes around the attribute and use `"%s"` with `html.EscapeString(...)` -- silencing `sprintfQuotedString` here means switching escaper, not switching verb
- never wrap `%s` in single quotes in a format string (`'%s'`) for human messages; use `%q` instead
- when the operand is already a `string` or a `Stringer` whose `String()` is known to be safe (e.g. `appdef.QName.String()`, `appdef.AppQName.String()`, integer-derived ids), `%s` without escaping is allowed for JSON / human-message / Go-quoted contexts; HTML and URL contexts still require the context-appropriate escaper even for safe Stringers, because the issue there is structural (which characters terminate the attribute / path segment), not value sanitization
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

bad -- HTML attribute with `%q` (Go-string escaping, NOT HTML escaping; embedded `"` still breaks out of the attribute):

```go
html := fmt.Sprintf(`<a href=%q>%s</a>`, ref, text)
```

good -- `"%s"` with `html.EscapeString` for both the attribute value and the text node:

```go
html := fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(ref), html.EscapeString(text))
```

good -- URL embedded in `href`: URL-escape the user parts first, then HTML-escape the whole URL:

```go
ref := "/items/" + url.PathEscape(itemID) + "?q=" + url.QueryEscape(query)
html := fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(ref), html.EscapeString(text))
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

1. Classify the embedding context first (JSON, URL path, URL query, host:port, file path, shell, SQL, HTML attribute, HTML text node, URL inside HTML attribute, human message, Go-quoted literal); never skip this step
2. Pick the verb and helper from "Decision by embedding context" above for that exact context
3. Verify the verb/escaper combination matches the chosen context: in JSON / Go-quoted contexts the format string must contain no `"%s"` (use `%q` instead); in HTML contexts the format string must keep `"%s"` AND wrap the operand in `html.EscapeString(...)` (using `%q` here is a bug); in URL contexts the operand must go through `url.PathEscape` / `url.QueryEscape`; in human messages the format string must contain no `'%s'` (use `%q` instead)
4. If a `sprintfQuotedString` linter hit is in an HTML context, fix it by switching the escaper to `html.EscapeString` and keeping `"%s"`; do NOT "fix" it by switching `"%s"` to `%q`
5. If you wrote `fmt.Sprintf("%s", x)` or `fmt.Sprint(x)`, drop the wrapper
6. If the operand is an `int*`, use `strconv` instead of `fmt.Sprintf("%d", ...)`
