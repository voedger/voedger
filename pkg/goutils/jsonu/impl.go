/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package jsonu

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Jprintf works like fmt.Sprintf but JSON-escapes string-like arguments:
// string, named ~string types, and values implementing fmt.Stringer. All
// other arguments are forwarded to fmt.Sprintf unchanged, so verbs such as
// %d, %t, %g work as usual.
//
// Use %s (or %v) for JSON string positions, and provide the surrounding
// double quotes in the format string yourself, for example:
//
//	Jprintf(`{"name":"%s"}`, name)
//
// The escaped content is emitted without surrounding quotes. Do NOT use %q
// on string-like arguments: the content is already JSON-escaped, so Go-quoting
// it again produces double-escaped, corrupted output.
//
// fmt.Stringer arguments are formatted lazily, so flags, width and precision
// (for example %-10.3s) are honored.
//
// This is stricter than fmt.Sprintf with %q. Go quoting can emit escapes that
// are invalid inside JSON strings, for example \v for a vertical tab and \xNN
// for invalid UTF-8 bytes. json.Marshal emits JSON-compatible escapes such as
// \u000b and coerces invalid UTF-8 to \ufffd.
func Jprintf(format string, args ...any) string {
	jargs := make([]any, len(args))
	for i, arg := range args {
		jargs[i] = jprintfArg(arg)
	}
	return fmt.Sprintf(format, jargs...)
}

func jprintfArg(arg any) any {
	switch v := arg.(type) {
	case string:
		return jsonEscapedStringContent(v)
	case fmt.Stringer:
		return jsonStringer{v}
	case nil:
		return arg
	}

	value := reflect.ValueOf(arg)
	if value.Kind() == reflect.String {
		// named string type (~string) without a String() method
		return jsonEscapedStringContent(value.String())
	}
	return arg
}

type jsonStringer struct {
	fmt.Stringer
}

func (s jsonStringer) Format(state fmt.State, verb rune) {
	arg := any(s.Stringer)
	if verb == 's' || verb == 'q' || verb == 'v' {
		arg = jsonEscapedStringContent(s.String())
	}
	fmt.Fprintf(state, fmtStateFormat(state, verb), arg)
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
