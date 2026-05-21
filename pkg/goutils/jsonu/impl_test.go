/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package jsonu

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

type testStringer struct {
	value string
}

func (s testStringer) String() string {
	return s.value
}

type testString string

type testIntStringer int32

func (s testIntStringer) String() string {
	return "testIntStringer"
}

func TestJprintf(t *testing.T) {
	require := require.New(t)

	t.Run("escapes string args without adding outer quotes", func(t *testing.T) {
		value := "quote: \", slash: \\, newline: \n, vtab: \v, nul: \x00"

		got := Jprintf(`{"value":"%s"}`, value)

		var decoded struct {
			Value string `json:"value"`
		}
		require.NoError(json.Unmarshal([]byte(got), &decoded))
		require.Equal(value, decoded.Value)
	})

	t.Run("escapes stringer args using String result", func(t *testing.T) {
		got := Jprintf(`{"value":"%s"}`, testStringer{value: `a"b\c`})

		require.JSONEq(`{"value":"a\"b\\c"}`, got)
	})

	t.Run("escapes stringer args with v verb", func(t *testing.T) {
		got := Jprintf(`{"value":"%v"}`, testStringer{value: `a"b`})

		require.JSONEq(`{"value":"a\"b"}`, got)
	})

	t.Run("escapes stringer args with q verb", func(t *testing.T) {
		got := Jprintf(`{"value":%q}`, testStringer{value: `a"b`})

		require.JSONEq(`{"value":"a\\\"b"}`, got)
	})

	t.Run("escapes named string args", func(t *testing.T) {
		got := Jprintf(`{"value":"%s"}`, testString(`a"b`))

		require.JSONEq(`{"value":"a\"b"}`, got)
	})

	t.Run("keeps non-string args compatible with fmt.Sprintf", func(t *testing.T) {
		got := Jprintf(`{"count":%d,"ok":%t,"nil":%v}`, 10, true, nil)

		require.Equal(fmt.Sprintf(`{"count":%d,"ok":%t,"nil":%v}`, 10, true, nil), got)
	})

	t.Run("keeps stringer args compatible with non-string verbs", func(t *testing.T) {
		got := Jprintf(`{"state":%d,"name":"%s"}`, testIntStringer(2), testStringer{value: `a"b`})

		require.JSONEq(`{"state":2,"name":"a\"b"}`, got)
	})

	t.Run("empty string produces empty content", func(t *testing.T) {
		require.JSONEq(`{"v":""}`, Jprintf(`{"v":"%s"}`, ""))
	})

	t.Run("preserves unicode characters", func(t *testing.T) {
		require.JSONEq(`{"v":"héllo🙂"}`, Jprintf(`{"v":"%s"}`, "héllo🙂"))
	})

	t.Run("does not escape forward slash", func(t *testing.T) {
		require.JSONEq(`{"v":"a/b"}`, Jprintf(`{"v":"%s"}`, "a/b"))
	})

	t.Run("escapes html-sensitive characters", func(t *testing.T) {
		require.JSONEq(`{"v":"\u003cb\u003e\u0026\u003c/b\u003e"}`, Jprintf(`{"v":"%s"}`, "<b>&</b>"))
	})

	t.Run("escapes line and paragraph separators", func(t *testing.T) {
		require.JSONEq(`{"v":"a\u2028b\u2029c"}`, Jprintf(`{"v":"%s"}`, "a\u2028b\u2029c"))
	})

	t.Run("escapes control characters", func(t *testing.T) {
		require.JSONEq(`{"v":"\u0000\u0001\b\t\n\r\u001f"}`,
			Jprintf(`{"v":"%s"}`, "\x00\x01\x08\x09\x0a\x0d\x1f"))
	})

	t.Run("replaces invalid utf-8 with replacement character", func(t *testing.T) {
		require.JSONEq(`{"v":"a\ufffdb"}`, Jprintf(`{"v":"%s"}`, "a\xffb"))
	})

	t.Run("formats multiple mixed args", func(t *testing.T) {
		require.JSONEq(`{"s":"a\"b","n":42,"b":true,"f":1.5}`,
			Jprintf(`{"s":"%s","n":%d,"b":%t,"f":%g}`, `a"b`, 42, true, 1.5))
	})

	t.Run("escapes explicitly indexed args", func(t *testing.T) {
		require.JSONEq(`{"name":"a\"1","state":2}`,
			Jprintf(`{"name":"%[2]s","state":%[1]d}`, testIntStringer(2), testStringer{value: `a"1`}))
	})

	t.Run("escapes explicitly indexed args with star width", func(t *testing.T) {
		require.JSONEq(`{"name":" a\"1","state":2}`,
			Jprintf(`{"name":"%[3]*[2]s","state":%[1]d}`, testIntStringer(2), testStringer{value: `a"1`}, 5))
	})

	t.Run("no args and no placeholders", func(t *testing.T) {
		require.JSONEq(`{"v":"static"}`, Jprintf(`{"v":"static"}`))
	})

	t.Run("%q double-escapes already-escaped string content", func(t *testing.T) {
		got := Jprintf(`{"v":%q}`, `a"b`)
		require.JSONEq(`{"v":"a\\\"b"}`, got)
		var decoded struct {
			V string `json:"v"`
		}
		require.NoError(json.Unmarshal([]byte(got), &decoded))
		require.Equal(`a\"b`, decoded.V)
	})

	t.Run("%q on non-string arg matches fmt.Sprintf", func(t *testing.T) {
		require.Equal(fmt.Sprintf(`{"v":%q}`, 'A'), Jprintf(`{"v":%q}`, 'A'))
	})

	t.Run("respects width, precision and flags on Stringer args", func(t *testing.T) {
		require.Equal(`{"v":"abc       "}`, Jprintf(`{"v":"%-10.3s"}`, testStringer{value: "abcdef"}))
	})

	t.Run("nil-typed pointer Stringer yields fmt PANIC placeholder", func(t *testing.T) {
		var s *testStringer
		got := Jprintf(`{"v":"%s"}`, s)
		require.Contains(got, "%!s(PANIC=")
		require.Contains(got, "nil *testStringer pointer")
	})
}
