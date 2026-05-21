/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package jsonu

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Jprintf works like fmt.Sprintf, but JSON-escapes string and fmt.Stringer arguments.
//
// Use it with JSON string placeholders that already have surrounding double quotes
// in the format string, for example: Jprintf(`{"name":"%s"}`, name).
//
// This is stricter than fmt.Sprintf with %q. Go quoting can emit escapes that are
// invalid inside JSON strings, for example \v for a vertical tab and \xNN for
// invalid UTF-8 bytes. json.Marshal emits JSON-compatible escapes such as \u000b
// and coerces invalid UTF-8 to \ufffd.
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
		return jsonEscapedStringContent(v.String())
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

func jsonEscapedStringContent(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// notest
		panic(err)
	}
	return string(b[1 : len(b)-1])
}
