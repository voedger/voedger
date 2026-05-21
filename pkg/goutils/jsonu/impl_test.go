/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package jsonu

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestJprintf_BasicUsage(t *testing.T) {
	require := require.New(t)
	name := "He said \"hi\"\v"
	err := errors.New("disk\tfull")

	body := Jprintf(`{"name":%q,"err":%q,"count":%d}`, name, err, 3)

	require.JSONEq(`{"name":"He said \"hi\"\u000b","err":"disk\tfull","count":3}`, body)
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

	t.Run("emits complete JSON string literal for stringer args with q verb", func(t *testing.T) {
		got := Jprintf(`{"value":%q}`, testStringer{value: `a"b`})

		require.JSONEq(`{"value":"a\"b"}`, got)
	})

	t.Run("escapes error args using Error result", func(t *testing.T) {
		got := Jprintf(`{"err":"%s"}`, errors.New(`bad "quote" and \slash`))

		require.JSONEq(`{"err":"bad \"quote\" and \\slash"}`, got)
	})

	t.Run("escapes error args with v verb", func(t *testing.T) {
		got := Jprintf(`{"err":"%v"}`, errors.New("line\vbreak"))

		require.JSONEq(`{"err":"line\u000bbreak"}`, got)
	})

	t.Run("emits complete JSON string literal for error args with q verb", func(t *testing.T) {
		got := Jprintf(`{"err":%q}`, errors.New(`a"b`))

		require.JSONEq(`{"err":"a\"b"}`, got)
	})

	t.Run("escapes wrapped error from fmt.Errorf", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", errors.New(`inner "msg"`))
		got := Jprintf(`{"err":%q}`, wrapped)

		require.JSONEq(`{"err":"outer: inner \"msg\""}`, got)
	})

	t.Run("custom error type with special chars", func(t *testing.T) {
		got := Jprintf(`{"err":%q}`, testError{value: "tab:\t,nl:\n,vt:\v"})

		require.JSONEq(`{"err":"tab:\t,nl:\n,vt:\u000b"}`, got)
	})

	t.Run("error takes precedence over Stringer when type implements both", func(t *testing.T) {
		// mirrors fmt's precedence: Error() wins over String() for %s/%v/%q
		arg := testStringerError{stringResult: "from-String", errorResult: `from-"Error"`}
		got := Jprintf(`{"v":%q}`, arg)

		require.JSONEq(`{"v":"from-\"Error\""}`, got)
	})

	t.Run("non-string verb on error arg passes through to fmt", func(t *testing.T) {
		err := errors.New("x")
		require.Equal(fmt.Sprintf(`%d`, err), Jprintf(`%d`, err))
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

	t.Run("%q on string emits a complete JSON string literal", func(t *testing.T) {
		got := Jprintf(`{"v":%q}`, `a"b`)
		require.JSONEq(`{"v":"a\"b"}`, got)
		var decoded struct {
			V string `json:"v"`
		}
		require.NoError(json.Unmarshal([]byte(got), &decoded))
		require.Equal(`a"b`, decoded.V)
	})

	t.Run("%q on non-string arg matches fmt.Sprintf", func(t *testing.T) {
		require.Equal(fmt.Sprintf(`{"v":%q}`, 'A'), Jprintf(`{"v":%q}`, 'A'))
	})

	t.Run("respects width, precision and flags on Stringer args", func(t *testing.T) {
		require.JSONEq(`{"v":"abc       "}`, Jprintf(`{"v":"%-10.3s"}`, testStringer{value: "abcdef"}))
	})

	t.Run("non-string verb on string arg passes through to fmt", func(t *testing.T) {
		require.Equal(fmt.Sprintf(`{"v":%x}`, "ab"), Jprintf(`{"v":%x}`, "ab"))
	})

	t.Run("%q honors width by padding the quoted output", func(t *testing.T) {
		require.Equal(`     "abc"`, Jprintf(`%10q`, "abc"))
		require.Equal(`"abc"     `, Jprintf(`%-10q`, testStringer{value: "abc"}))
	})

	qname := testStringer{value: "app.Doc"}
	name := "line\vwith \"quotes\""
	type readmePayload struct {
		QName string `json:"qname"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	t.Run("%q in README example round-trips correctly", func(t *testing.T) {
		got := Jprintf(`{"qname":%q,"name":%q,"count":%d}`, qname, name, 3)
		var decoded readmePayload
		require.NoError(json.Unmarshal([]byte(got), &decoded))
		require.Equal("app.Doc", decoded.QName)
		require.Equal(name, decoded.Name)
		require.Equal(3, decoded.Count)
	})

	t.Run("%s in README example round-trips correctly", func(t *testing.T) {
		got := Jprintf(`{"qname":"%s","name":"%s","count":%d}`, qname, name, 3)
		var decoded readmePayload
		require.NoError(json.Unmarshal([]byte(got), &decoded))
		require.Equal("app.Doc", decoded.QName)
		require.Equal(name, decoded.Name)
		require.Equal(3, decoded.Count)
	})

	t.Run("nil-typed pointer Stringer yields fmt PANIC placeholder", func(t *testing.T) {
		var s *testStringer
		got := Jprintf(`{"v":"%s"}`, s)
		require.Contains(got, "%!s(PANIC=")
		require.Contains(got, "nil *testStringer pointer")
	})
}

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

type testError struct {
	value string
}

func (e testError) Error() string {
	return e.value
}

type testStringerError struct {
	stringResult string
	errorResult  string
}

func (e testStringerError) String() string { return e.stringResult }
func (e testStringerError) Error() string  { return e.errorResult }
