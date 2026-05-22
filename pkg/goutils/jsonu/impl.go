/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package jsonu

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Jprintf works like fmt.Sprintf but JSON-escapes string-like arguments:
// string, named ~string types, values implementing fmt.Stringer, and values
// implementing the error interface. All other arguments are forwarded to
// fmt.Sprintf unchanged, so verbs such as %d, %t, %g work as usual.
//
// When a type implements both error and fmt.Stringer, Error() is preferred
// (mirroring fmt.Sprintf precedence).
//
// Verbs for string-like arguments:
//
//   - %s and %v emit the JSON-escaped content without surrounding quotes.
//     The caller is expected to provide the quotes in the template:
//
//     Jprintf(`{"name":"%s"}`, name)
//
//   - %q emits a complete JSON string literal (escaped content wrapped in
//     double quotes). No surrounding quotes are needed in the template:
//
//     Jprintf(`{"name":%q}`, name)
//
// String-like arguments are formatted lazily, so flags and width are honored
// (for example %-10.3s on a Stringer, or %10q which pads the whole quoted
// output to width 10). Precision on %q is byte-counted over the quoted
// result, which differs from fmt.Sprintf's character-counted precision.
//
// This is stricter than fmt.Sprintf with %q. Go quoting can emit escapes that
// are invalid inside JSON strings, for example \v for a vertical tab and \xNN
// for invalid UTF-8 bytes. json.Marshal emits JSON-compatible escapes such as
// \u000b and coerces invalid UTF-8 to \ufffd.
func Jprintf(format string, args ...any) string {
	return fmt.Sprintf(format, jprintfArgs(args)...)
}

// Jfprintf writes the Jprintf result to w. Same verbs and escaping rules as Jprintf.
func Jfprintf(w io.Writer, format string, args ...any) (n int, err error) {
	return fmt.Fprintf(w, format, jprintfArgs(args)...)
}

func jprintfArgs(args []any) []any {
	jargs := make([]any, len(args))
	for i, arg := range args {
		jargs[i] = jprintfArg(arg)
	}
	return jargs
}

func jprintfArg(arg any) any {
	switch v := arg.(type) {
	case string:
		return jsonString(v)
	case error:
		return jsonError{v}
	case fmt.Stringer:
		return jsonStringer{v}
	case nil:
		return arg
	}

	value := reflect.ValueOf(arg)
	if value.Kind() == reflect.String {
		// named string type (~string) without a String() method
		return jsonString(value.String())
	}
	return arg
}

func (s jsonString) Format(state fmt.State, verb rune) {
	if verb == 's' || verb == 'v' || verb == 'q' {
		formatJSONString(state, verb, string(s))
		return
	}
	fmt.Fprintf(state, fmtStateFormat(state, verb), string(s))
}

func (s jsonStringer) Format(state fmt.State, verb rune) {
	if verb == 's' || verb == 'v' || verb == 'q' {
		formatJSONString(state, verb, s.String())
		return
	}
	fmt.Fprintf(state, fmtStateFormat(state, verb), s.Stringer)
}

func (e jsonError) Format(state fmt.State, verb rune) {
	if verb == 's' || verb == 'v' || verb == 'q' {
		formatJSONString(state, verb, e.Error())
		return
	}
	fmt.Fprintf(state, fmtStateFormat(state, verb), e.error)
}

func formatJSONString(state fmt.State, verb rune, s string) {
	escaped := jsonEscapedStringContent(s)
	if verb == 'q' {
		fmt.Fprintf(state, fmtStateFormat(state, 's'), `"`+escaped+`"`)
		return
	}
	fmt.Fprintf(state, fmtStateFormat(state, verb), escaped)
}

func fmtStateFormat(state fmt.State, verb rune) string {
	var b strings.Builder
	b.WriteByte('%')
	for _, flag := range "#+-0 " {
		if state.Flag(int(flag)) {
			b.WriteRune(flag)
		}
	}
	if width, ok := state.Width(); ok {
		b.WriteString(strconv.Itoa(width))
	}
	if precision, ok := state.Precision(); ok {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(precision))
	}
	b.WriteRune(verb)
	return b.String()
}

func jsonEscapedStringContent(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// notest
		panic(err)
	}
	return string(b[1 : len(b)-1])
}
